package repository

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/agridispatch/agridispatch/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBSeq int64

// newTestDB 构造独立的内存 SQLite 并迁移派单闭环涉及的表。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:repotest%d?mode=memory&cache=shared", atomic.AddInt64(&testDBSeq, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Machine{},
		&model.FarmTask{},
		&model.Driver{},
		&model.DispatchAttempt{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

// seedScenario 构造标准测试数据，可对单个任务/农机/驾驶员做覆盖调整。
func seedScenario(t *testing.T, db *gorm.DB, mutate func(machines *[]model.Machine, drivers *[]model.Driver, tasks *[]model.FarmTask)) {
	t.Helper()
	machines := []model.Machine{
		{ID: "m1", Code: "M-A", Status: "空闲", MaintenanceHours: 20, TaskID: "", CurrentTask: ""},
		{ID: "m2", Code: "M-BUSY", Status: "作业中", MaintenanceHours: 100, TaskID: "other-task", CurrentTask: "其他任务"},
		{ID: "m3", Code: "M-LOW", Status: "空闲", MaintenanceHours: 2, TaskID: ""},
	}
	drivers := []model.Driver{
		{ID: "d1", Name: "张三", Status: "在岗", WorkStatus: "空闲", TaskID: ""},
		{ID: "d2", Name: "李四", Status: "休息", WorkStatus: "空闲", TaskID: ""},
		{ID: "d3", Name: "王五", Status: "在岗", WorkStatus: "作业中", TaskID: "other-task"},
	}
	tasks := []model.FarmTask{
		{ID: "ok", Type: "耕地", Field: "一号田", EstimatedHours: 8, Status: "待派单", RecommendedMachine: "M-A", RecommendedDriver: "张三"},
		{ID: "busy", Type: "耕地", Field: "二号田", EstimatedHours: 8, Status: "待派单", RecommendedMachine: "M-BUSY", RecommendedDriver: "张三"},
		{ID: "low", Type: "耕地", Field: "三号田", EstimatedHours: 8, Status: "待派单", RecommendedMachine: "M-LOW", RecommendedDriver: "张三"},
		{ID: "off", Type: "耕地", Field: "四号田", EstimatedHours: 8, Status: "待派单", RecommendedMachine: "M-A", RecommendedDriver: "李四"},
		{ID: "drvbusy", Type: "耕地", Field: "五号田", EstimatedHours: 8, Status: "待派单", RecommendedMachine: "M-A", RecommendedDriver: "王五"},
	}
	if mutate != nil {
		mutate(&machines, &drivers, &tasks)
	}
	if err := db.Create(&machines).Error; err != nil {
		t.Fatalf("seed machines: %v", err)
	}
	if err := db.Create(&drivers).Error; err != nil {
		t.Fatalf("seed drivers: %v", err)
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("seed tasks: %v", err)
	}
}

func mustFindMachine(t *testing.T, db *gorm.DB, code string) model.Machine {
	t.Helper()
	var m model.Machine
	if err := db.First(&m, "code = ?", code).Error; err != nil {
		t.Fatalf("find machine %s: %v", code, err)
	}
	return m
}

func mustFindDriver(t *testing.T, db *gorm.DB, name string) model.Driver {
	t.Helper()
	var d model.Driver
	if err := db.First(&d, "name = ?", name).Error; err != nil {
		t.Fatalf("find driver %s: %v", name, err)
	}
	return d
}

func mustFindTask(t *testing.T, db *gorm.DB, id string) model.FarmTask {
	t.Helper()
	var task model.FarmTask
	if err := db.First(&task, "id = ?", id).Error; err != nil {
		t.Fatalf("find task %s: %v", id, err)
	}
	return task
}
