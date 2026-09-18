package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

var dbSeq int64

func newHTTPServer(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:hdtest%d?mode=memory&cache=shared", atomic.AddInt64(&dbSeq, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Machine{}, &model.FarmTask{}, &model.Driver{}, &model.DispatchAttempt{},
		&model.DashboardItem{}, &model.TrackPoint{}, &model.WorkRecord{}, &model.MaintenanceReminder{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create([]model.Machine{
		{ID: "m1", Code: "M-OK", Status: "空闲", MaintenanceHours: 20},
		{ID: "m2", Code: "M-LOW", Status: "空闲", MaintenanceHours: 1},
		{ID: "m3", Code: "M-FOROFF", Status: "空闲", MaintenanceHours: 20},
	}).Error; err != nil {
		t.Fatalf("seed machines: %v", err)
	}
	if err := db.Create([]model.Driver{
		{ID: "d1", Name: "甲", Status: "在岗", WorkStatus: "空闲"},
		{ID: "d2", Name: "乙", Status: "休息", WorkStatus: "空闲"},
	}).Error; err != nil {
		t.Fatalf("seed drivers: %v", err)
	}
	if err := db.Create([]model.FarmTask{
		{ID: "t-ok", Type: "耕地", Field: "田1", EstimatedHours: 6, Status: "待派单", RecommendedMachine: "M-OK", RecommendedDriver: "甲"},
		{ID: "t-low", Type: "耕地", Field: "田2", EstimatedHours: 6, Status: "待派单", RecommendedMachine: "M-LOW", RecommendedDriver: "甲"},
		{ID: "t-off", Type: "耕地", Field: "田3", EstimatedHours: 6, Status: "待派单", RecommendedMachine: "M-FOROFF", RecommendedDriver: "乙"},
	}).Error; err != nil {
		t.Fatalf("seed tasks: %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	dashSvc := service.NewDashboardService(repository.NewDashboardRepository(db), rdb, log)
	dispatchSvc := service.NewDispatchService(repository.NewDispatchRepository(db), rdb, log)

	r := gin.New()
	h1 := NewDashboardHandler(dashSvc)
	h2 := NewDispatchHandler(dispatchSvc)
	r.GET("/api/v1/dashboard/overview", h1.Overview)
	r.POST("/api/v1/tasks/:id/dispatch", h2.Dispatch)
	r.POST("/api/v1/tasks/:id/cancel", h2.Cancel)
	return r, db
}

func do(t *testing.T, r http.Handler, method, path string) (int, apiBody) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body apiBody
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body)
	}
	return w.Code, body
}

// TestHTTPDispatchRejectAndPersist 保养不足返回 409，且 overview 能回读失败原因。
func TestHTTPDispatchRejectAndPersist(t *testing.T) {
	r, _ := newHTTPServer(t)

	status, body := do(t, r, http.MethodPost, "/api/v1/tasks/t-low/dispatch")
	if status != http.StatusConflict || body.Code != constants.CodeConflict {
		t.Fatalf("status=%d body=%+v", status, body)
	}
	if body.Message != constants.ReasonMaintenanceShort {
		t.Fatalf("message=%q", body.Message)
	}

	_, ov := do(t, r, http.MethodGet, "/api/v1/dashboard/overview")
	var data model.FarmOverview
	if err := json.Unmarshal(ov.Data, &data); err != nil {
		t.Fatalf("unmarshal overview: %v", err)
	}
	var task *model.FarmTask
	for i := range data.Tasks {
		if data.Tasks[i].ID == "t-low" {
			task = &data.Tasks[i]
		}
	}
	if task == nil {
		t.Fatal("task t-low missing in overview")
	}
	if task.Status != constants.TaskPending {
		t.Errorf("rejected task status=%s", task.Status)
	}
	if task.LastAttemptResult != constants.DispatchResultReject || task.LastAttemptReason != constants.ReasonMaintenanceShort {
		t.Errorf("last attempt = %s/%s", task.LastAttemptResult, task.LastAttemptReason)
	}
	found := false
	for _, a := range data.Attempts {
		if a.TaskID == "t-low" && a.Result == constants.DispatchResultReject {
			found = true
		}
	}
	if !found {
		t.Error("rejected attempt not returned in overview")
	}
}

// TestHTTPDispatchThenCancelLifecycle 成功派单 -> 重复派单 409 -> 取消释放 -> 重复取消幂等。
func TestHTTPDispatchThenCancelLifecycle(t *testing.T) {
	r, db := newHTTPServer(t)

	if status, body := do(t, r, http.MethodPost, "/api/v1/tasks/t-ok/dispatch"); status != http.StatusOK || body.Code != 0 {
		t.Fatalf("dispatch status=%d body=%+v", status, body)
	}
	// 重复派单冲突
	if status, _ := do(t, r, http.MethodPost, "/api/v1/tasks/t-ok/dispatch"); status != http.StatusConflict {
		t.Fatalf("duplicate dispatch status=%d, want 409", status)
	}
	// 驾驶员休息的任务仍拒绝，且不影响已占用的甲
	if status, body := do(t, r, http.MethodPost, "/api/v1/tasks/t-off/dispatch"); status != http.StatusConflict || body.Message != constants.ReasonDriverOffDuty {
		t.Fatalf("off-duty dispatch status=%d msg=%q", status, body.Message)
	}

	// 取消
	if status, body := do(t, r, http.MethodPost, "/api/v1/tasks/t-ok/cancel"); status != http.StatusOK || body.Code != 0 {
		t.Fatalf("cancel status=%d body=%+v", status, body)
	}
	var machine model.Machine
	_ = db.First(&machine, "code = ?", "M-OK").Error
	if machine.Status != constants.MachineIdle || machine.TaskID != "" {
		t.Errorf("machine not released: %+v", machine)
	}
	// 再次取消：幂等空操作（仍 200，released=false）
	_, body := do(t, r, http.MethodPost, "/api/v1/tasks/t-ok/cancel")
	var res struct {
		Released bool `json:"released"`
	}
	_ = json.Unmarshal(body.Data, &res)
	if res.Released {
		t.Error("second cancel must be idempotent noop")
	}
}
