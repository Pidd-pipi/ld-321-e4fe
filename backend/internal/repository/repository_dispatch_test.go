package repository

import (
	"errors"
	"sync"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
)

// TestDispatchSuccess 条件齐备：任务/农机/驾驶员一起标记作业中。
func TestDispatchSuccess(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	res, err := repo.Dispatch("ok")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if res.MachineCode != "M-A" || res.DriverName != "张三" {
		t.Fatalf("unexpected binding: %+v", res)
	}

	task := mustFindTask(t, db, "ok")
	machine := mustFindMachine(t, db, "M-A")
	driver := mustFindDriver(t, db, "张三")
	if task.Status != constants.TaskDispatched {
		t.Errorf("task status=%s", task.Status)
	}
	if machine.Status != constants.MachineWorking || machine.TaskID != "ok" || machine.CurrentTask == "" {
		t.Errorf("machine not occupied: %+v", machine)
	}
	if driver.WorkStatus != constants.DriverWorking || driver.TaskID != "ok" {
		t.Errorf("driver not occupied: %+v", driver)
	}
}

// TestDispatchRejections 每种不满足条件都整单拒绝，且不留下任何单向占用，并落库原因。
func TestDispatchRejections(t *testing.T) {
	cases := []struct {
		name   string
		taskID string
		reason string
	}{
		{"农机非空闲", "busy", constants.ReasonMachineNotIdle},
		{"保养工时不足", "low", constants.ReasonMaintenanceShort},
		{"驾驶员不在岗", "off", constants.ReasonDriverOffDuty},
		{"驾驶员已被占用", "drvbusy", constants.ReasonDriverBusy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			seedScenario(t, db, nil)
			repo := NewDispatchRepository(db)

			_, err := repo.Dispatch(tc.taskID)
			var rejected *DispatchRejected
			if !errors.As(err, &rejected) || rejected.Reason != tc.reason {
				t.Fatalf("want reject %q, got %v", tc.reason, err)
			}

			// 任务仍是待派单。
			task := mustFindTask(t, db, tc.taskID)
			if task.Status != constants.TaskPending {
				t.Errorf("task status changed to %s", task.Status)
			}
			// M-A 必须保持空闲（除 busy 用例本就不涉及外，其余用例绝不能被单向占用）。
			ma := mustFindMachine(t, db, "M-A")
			if ma.Status != constants.MachineIdle || ma.TaskID != "" {
				t.Errorf("M-A leaked occupation: %+v", ma)
			}
			zhangsan := mustFindDriver(t, db, "张三")
			if zhangsan.WorkStatus != constants.DriverIdle || zhangsan.TaskID != "" {
				t.Errorf("张三 leaked occupation: %+v", zhangsan)
			}
			// 失败原因已落库到任务与尝试表（刷新后可回读）。
			if task.LastAttemptResult != constants.DispatchResultReject || task.LastAttemptReason != tc.reason {
				t.Errorf("task last attempt = %s/%s", task.LastAttemptResult, task.LastAttemptReason)
			}
			var attempts []model.DispatchAttempt
			if err := db.Where("task_id = ? AND action = ?", tc.taskID, constants.DispatchActionDispatch).Find(&attempts).Error; err != nil {
				t.Fatalf("find attempts: %v", err)
			}
			if len(attempts) != 1 || attempts[0].Result != constants.DispatchResultReject || attempts[0].Reason != tc.reason {
				t.Fatalf("attempts = %+v", attempts)
			}
		})
	}
}

// TestDispatchAlreadyDispatched 重复派单被拒绝。
func TestDispatchAlreadyDispatched(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	if _, err := repo.Dispatch("ok"); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	if _, err := repo.Dispatch("ok"); err == nil {
		t.Fatal("second dispatch must be rejected")
	}
	task := mustFindTask(t, db, "ok")
	if task.Status != constants.TaskDispatched {
		t.Errorf("task status=%s", task.Status)
	}
}

// TestCancelReleasesAll 取消同步释放任务/农机/驾驶员。
func TestCancelReleasesAll(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	if _, err := repo.Dispatch("ok"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	res, err := repo.Cancel("ok")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !res.Released {
		t.Fatal("cancel should release once")
	}
	task := mustFindTask(t, db, "ok")
	machine := mustFindMachine(t, db, "M-A")
	driver := mustFindDriver(t, db, "张三")
	if task.Status != constants.TaskPending {
		t.Errorf("task status=%s, want 待派单", task.Status)
	}
	if machine.Status != constants.MachineIdle || machine.TaskID != "" || machine.CurrentTask != "" {
		t.Errorf("machine not released: %+v", machine)
	}
	if driver.WorkStatus != constants.DriverIdle || driver.TaskID != "" {
		t.Errorf("driver not released: %+v", driver)
	}
}

// TestCancelIdempotent 重复取消只释放一次，第二次为幂等空操作。
func TestCancelIdempotent(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	if _, err := repo.Dispatch("ok"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := repo.Cancel("ok"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	res2, err := repo.Cancel("ok")
	if err != nil {
		t.Fatalf("second cancel: %v", err)
	}
	if res2.Released {
		t.Fatal("second cancel must not release again")
	}
	var released int64
	db.Model(&model.DispatchAttempt{}).
		Where("task_id = ? AND action = ? AND result = ?", "ok", constants.DispatchActionCancel, constants.DispatchResultRelease).
		Count(&released)
	if released != 1 {
		t.Errorf("released records=%d, want 1", released)
	}
}

// TestDispatchConcurrentOnce 并发派单同一任务只能成功一次，且占用关系一致。
func TestDispatchConcurrentOnce(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	const n = 30
	var wg sync.WaitGroup
	var success, rejected int64
	var mu sync.Mutex
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.Dispatch("ok")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				rejected++
			}
		}()
	}
	close(start)
	wg.Wait()

	if success != 1 || rejected != n-1 {
		t.Fatalf("success=%d rejected=%d, want 1/%d", success, rejected, n-1)
	}
	machine := mustFindMachine(t, db, "M-A")
	driver := mustFindDriver(t, db, "张三")
	if machine.TaskID != "ok" || driver.TaskID != "ok" {
		t.Fatalf("binding inconsistent: machine=%+v driver=%+v", machine, driver)
	}
}

// TestCancelConcurrentOnce 并发取消同一任务只真正释放一次。
func TestCancelConcurrentOnce(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, nil)
	repo := NewDispatchRepository(db)

	if _, err := repo.Dispatch("ok"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	const n = 30
	var wg sync.WaitGroup
	var released int64
	var mu sync.Mutex
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			res, err := repo.Cancel("ok")
			if err != nil {
				t.Errorf("cancel err: %v", err)
				return
			}
			if res.Released {
				mu.Lock()
				released++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if released != 1 {
		t.Fatalf("released=%d, want 1", released)
	}
	machine := mustFindMachine(t, db, "M-A")
	driver := mustFindDriver(t, db, "张三")
	if machine.Status != constants.MachineIdle || machine.TaskID != "" ||
		driver.WorkStatus != constants.DriverIdle || driver.TaskID != "" {
		t.Fatalf("release inconsistent: machine=%+v driver=%+v", machine, driver)
	}
}

// TestTwoTasksSameResource 同机同人只能被一个任务占用，第二单整单拒绝。
func TestTwoTasksSameResource(t *testing.T) {
	db := newTestDB(t)
	seedScenario(t, db, func(machines *[]model.Machine, drivers *[]model.Driver, tasks *[]model.FarmTask) {
		*tasks = append(*tasks, model.FarmTask{
			ID: "ok2", Type: "播种", Field: "六号田", EstimatedHours: 4,
			Status: "待派单", RecommendedMachine: "M-A", RecommendedDriver: "张三",
		})
	})
	repo := NewDispatchRepository(db)

	if _, err := repo.Dispatch("ok"); err != nil {
		t.Fatalf("dispatch ok: %v", err)
	}
	if _, err := repo.Dispatch("ok2"); err == nil {
		t.Fatal("dispatch ok2 must reject when resource already occupied")
	}
	if task := mustFindTask(t, db, "ok2"); task.Status != constants.TaskPending {
		t.Errorf("ok2 status=%s, want 待派单", task.Status)
	}
}
