package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DispatchRepository 派单预占闭环的数据访问。
//
// 并发策略（双保险）：
//  1. 进程内 mu 串行化派单/取消关键区，单实例下并发请求只会有一个进入事务；
//  2. 事务内按固定顺序（任务 → 农机 → 驾驶员）SELECT ... FOR UPDATE 加行锁，
//     并用条件式 UPDATE（仅当状态仍满足时才生效）防止多实例部署下的交叉占用。
type DispatchRepository struct {
	db *gorm.DB
	mu sync.Mutex
}

func NewDispatchRepository(db *gorm.DB) *DispatchRepository {
	return &DispatchRepository{db: db}
}

// DispatchResult 派单事务结果。
type DispatchResult struct {
	TaskID      string
	MachineCode string
	DriverName  string
}

// Dispatch 派单预占：在单事务内校验并同时占用任务、农机、驾驶员。
// 任一前置条件不满足时，仅写入一条 rejected 尝试记录，不修改任何占用状态。
func (r *DispatchRepository) Dispatch(taskID string) (*DispatchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 先做无锁预检：不满足的硬性条件直接拒绝并留痕，不进入写事务。
	task, machine, driver, reason, err := r.precheck(taskID)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		if recErr := r.recordAttempt(task, bindingMachine(task, machine), bindingDriver(task, driver), constants.DispatchActionDispatch, constants.DispatchResultReject, reason); recErr != nil {
			return nil, recErr
		}
		return nil, ErrDispatchRejected(reason)
	}

	// 预检通过：进入加锁事务做最终占用（锁内状态可能已被并发请求改变）。
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		var locked model.FarmTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", task.ID).Error; err != nil {
			return fmt.Errorf("lock task: %w", err)
		}
		var lockedMachine model.Machine
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedMachine, "code = ?", machine.Code).Error; err != nil {
			return fmt.Errorf("lock machine: %w", err)
		}
		var lockedDriver model.Driver
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedDriver, "id = ?", driver.ID).Error; err != nil {
			return fmt.Errorf("lock driver: %w", err)
		}

		if reason := validateLocked(locked, lockedMachine, lockedDriver); reason != "" {
			return ErrDispatchRejected(reason)
		}

		now := time.Now()
		res := tx.Model(&model.Machine{}).
			Where("code = ? AND status = ?", machine.Code, constants.MachineIdle).
			Updates(map[string]interface{}{
				"status":       constants.MachineWorking,
				"current_task": taskDisplay(locked),
				"task_id":      locked.ID,
			})
		if res.Error != nil {
			return fmt.Errorf("occupy machine: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrDispatchRejected(constants.ReasonMachineNotIdle)
		}

		res = tx.Model(&model.Driver{}).
			Where("id = ? AND status = ? AND work_status = ?", driver.ID, constants.DriverOnDuty, constants.DriverIdle).
			Updates(map[string]interface{}{
				"work_status": constants.DriverWorking,
				"task_id":     locked.ID,
			})
		if res.Error != nil {
			return fmt.Errorf("occupy driver: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrDispatchRejected(constants.ReasonDriverBusy)
		}

		res = tx.Model(&model.FarmTask{}).
			Where("id = ? AND status = ?", locked.ID, constants.TaskPending).
			Updates(map[string]interface{}{
				"status":              constants.TaskDispatched,
				"last_attempt_action": constants.DispatchActionDispatch,
				"last_attempt_result": constants.DispatchResultSuccess,
				"last_attempt_reason": "",
				"last_attempt_at":     now,
			})
		if res.Error != nil {
			return fmt.Errorf("occupy task: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrDispatchRejected(constants.ReasonTaskNotPending)
		}

		attempt := buildAttempt(locked, lockedMachine.Code, lockedDriver.Name, constants.DispatchActionDispatch, constants.DispatchResultSuccess, "", now)
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("record dispatch attempt: %w", err)
		}
		return nil
	}); err != nil {
		// 锁内复检失败属于业务拒绝：补写 rejected 留痕（事务已整体回滚，无任何占用残留）。
		var rej *DispatchRejected
		if errors.As(err, &rej) {
			if recErr := r.recordAttempt(task, bindingMachine(task, machine), bindingDriver(task, driver), constants.DispatchActionDispatch, constants.DispatchResultReject, rej.Reason); recErr != nil {
				return nil, recErr
			}
			return nil, rej
		}
		return nil, err
	}

	return &DispatchResult{TaskID: task.ID, MachineCode: machine.Code, DriverName: driver.Name}, nil
}

// CancelResult 取消派单结果。Released=false 表示此前已释放（幂等空操作）。
type CancelResult struct {
	TaskID   string
	Released bool
	Reason   string
}

// Cancel 取消派单：在单事务内同步释放任务、农机和驾驶员；重复/并发取消只生效一次。
func (r *DispatchRepository) Cancel(taskID string) (*CancelResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var task model.FarmTask
	err := r.db.First(&task, "id = ?", taskID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find task for cancel: %w", err)
	}
	if task.Status != constants.TaskDispatched {
		// 幂等：未处于作业中即无可释放占用，记录 noop 但不改状态。
		reason := constants.ReasonNotDispatched
		if err := r.recordAttempt(&task, task.RecommendedMachine, task.RecommendedDriver, constants.DispatchActionCancel, constants.DispatchResultNoop, reason); err != nil {
			return nil, err
		}
		return &CancelResult{TaskID: taskID, Released: false, Reason: reason}, nil
	}

	now := time.Now()
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		var locked model.FarmTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", taskID).Error; err != nil {
			return fmt.Errorf("lock task for cancel: %w", err)
		}
		if locked.Status != constants.TaskDispatched {
			return ErrDispatchRejected(constants.ReasonNotDispatched)
		}

		// 释放仅作用于当前任务实际绑定的农机/驾驶员，避免误释放他人占用。
		if err := tx.Model(&model.Machine{}).
			Where("task_id = ?", locked.ID).
			Updates(map[string]interface{}{
				"status":       constants.MachineIdle,
				"current_task": "",
				"task_id":      "",
			}).Error; err != nil {
			return fmt.Errorf("release machine: %w", err)
		}
		if err := tx.Model(&model.Driver{}).
			Where("task_id = ?", locked.ID).
			Updates(map[string]interface{}{
				"work_status": constants.DriverIdle,
				"task_id":     "",
			}).Error; err != nil {
			return fmt.Errorf("release driver: %w", err)
		}
		if err := tx.Model(&model.FarmTask{}).
			Where("id = ? AND status = ?", locked.ID, constants.TaskDispatched).
			Updates(map[string]interface{}{
				"status":              constants.TaskPending,
				"last_attempt_action": constants.DispatchActionCancel,
				"last_attempt_result": constants.DispatchResultRelease,
				"last_attempt_reason": "",
				"last_attempt_at":     now,
			}).Error; err != nil {
			return fmt.Errorf("release task: %w", err)
		}

		attempt := buildAttempt(locked, locked.RecommendedMachine, locked.RecommendedDriver, constants.DispatchActionCancel, constants.DispatchResultRelease, "", now)
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("record cancel attempt: %w", err)
		}
		return nil
	}); err != nil {
		var rej *DispatchRejected
		if errors.As(err, &rej) {
			if recErr := r.recordAttempt(&task, task.RecommendedMachine, task.RecommendedDriver, constants.DispatchActionCancel, constants.DispatchResultNoop, rej.Reason); recErr != nil {
				return nil, recErr
			}
			return &CancelResult{TaskID: taskID, Released: false, Reason: rej.Reason}, nil
		}
		return nil, err
	}

	return &CancelResult{TaskID: taskID, Released: true}, nil
}

// precheck 无锁预检，返回首个不满足条件的原因；reason 为空表示全部通过。
func (r *DispatchRepository) precheck(taskID string) (task *model.FarmTask, machine *model.Machine, driver *model.Driver, reason string, err error) {
	task = &model.FarmTask{}
	if err = r.db.First(task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, "", ErrNotFound
		}
		return nil, nil, nil, "", fmt.Errorf("find task: %w", err)
	}
	switch {
	case task.Status == constants.TaskDispatched:
		return task, nil, nil, constants.ReasonTaskNotPending, nil
	case task.Status == constants.TaskDone:
		return task, nil, nil, constants.ReasonTaskNotPending, nil
	case task.Status == constants.TaskCancelled:
		return task, nil, nil, constants.ReasonTaskCancelled, nil
	}
	if task.RecommendedMachine == "" || task.RecommendedDriver == "" {
		return task, nil, nil, constants.ReasonTaskMissingBinding, nil
	}

	machine = &model.Machine{}
	if err = r.db.First(machine, "code = ?", task.RecommendedMachine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return task, nil, nil, constants.ReasonMachineNotFound, nil
		}
		return nil, nil, nil, "", fmt.Errorf("find machine: %w", err)
	}
	driver = &model.Driver{}
	if err = r.db.First(driver, "name = ?", task.RecommendedDriver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return task, machine, nil, constants.ReasonDriverNotFound, nil
		}
		return nil, nil, nil, "", fmt.Errorf("find driver: %w", err)
	}

	if machine.Status != constants.MachineIdle {
		return task, machine, driver, constants.ReasonMachineNotIdle, nil
	}
	if machine.MaintenanceHours+constants.MaintenanceHoursEpsilon < task.EstimatedHours {
		return task, machine, driver, constants.ReasonMaintenanceShort, nil
	}
	if driver.Status != constants.DriverOnDuty {
		return task, machine, driver, constants.ReasonDriverOffDuty, nil
	}
	if driver.WorkStatus == constants.DriverWorking {
		return task, machine, driver, constants.ReasonDriverBusy, nil
	}
	return task, machine, driver, "", nil
}

// validateLocked 行锁内最终复检（与条件式 UPDATE 互为兜底）。
func validateLocked(task model.FarmTask, machine model.Machine, driver model.Driver) string {
	if task.Status != constants.TaskPending {
		return constants.ReasonTaskNotPending
	}
	if machine.Status != constants.MachineIdle {
		return constants.ReasonMachineNotIdle
	}
	if machine.MaintenanceHours+constants.MaintenanceHoursEpsilon < task.EstimatedHours {
		return constants.ReasonMaintenanceShort
	}
	if driver.Status != constants.DriverOnDuty {
		return constants.ReasonDriverOffDuty
	}
	if driver.WorkStatus == constants.DriverWorking {
		return constants.ReasonDriverBusy
	}
	return ""
}

// recordAttempt 独立事务写入尝试记录（拒绝/空操作不与状态回滚相互影响）。
func (r *DispatchRepository) recordAttempt(task *model.FarmTask, machineCode, driverName, action, result, reason string) error {
	attempt := model.DispatchAttempt{
		MachineCode: machineCode,
		DriverName:  driverName,
		Action:      action,
		Result:      result,
		Reason:      reason,
		CreatedAt:   time.Now(),
	}
	if task != nil {
		attempt.TaskID = task.ID
		attempt.TaskType = task.Type
		attempt.TaskField = task.Field
	}
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("create dispatch attempt: %w", err)
		}
		if task != nil {
			if err := tx.Model(&model.FarmTask{}).Where("id = ?", task.ID).
				Updates(map[string]interface{}{
					"last_attempt_action": action,
					"last_attempt_result": result,
					"last_attempt_reason": reason,
					"last_attempt_at":     attempt.CreatedAt,
				}).Error; err != nil {
				return fmt.Errorf("update task attempt: %w", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// 回写内存对象，供调用方即时读取。
	if task != nil {
		task.LastAttemptAction = action
		task.LastAttemptResult = result
		task.LastAttemptReason = reason
		task.LastAttemptAt = attempt.CreatedAt
	}
	return nil
}

// bindingMachine/ bindingDriver 在预检查对象可能缺失时回退到任务上记录的推荐绑定。
func bindingMachine(task *model.FarmTask, machine *model.Machine) string {
	if machine != nil {
		return machine.Code
	}
	if task != nil {
		return task.RecommendedMachine
	}
	return ""
}

func bindingDriver(task *model.FarmTask, driver *model.Driver) string {
	if driver != nil {
		return driver.Name
	}
	if task != nil {
		return task.RecommendedDriver
	}
	return ""
}

func buildAttempt(task model.FarmTask, machineCode, driverName, action, result, reason string, at time.Time) model.DispatchAttempt {
	return model.DispatchAttempt{
		TaskID:      task.ID,
		TaskType:    task.Type,
		TaskField:   task.Field,
		MachineCode: machineCode,
		DriverName:  driverName,
		Action:      action,
		Result:      result,
		Reason:      reason,
		CreatedAt:   at,
	}
}

func taskDisplay(t model.FarmTask) string {
	return fmt.Sprintf("%s %s", t.Type, t.Field)
}

// DispatchRejected 仓储层派单拒绝哨兵（内部类型，由 errors.As 识别）。
type DispatchRejected struct {
	Reason string
}

func (e *DispatchRejected) Error() string {
	return "dispatch rejected: " + e.Reason
}

// ErrDispatchRejected 构造派单拒绝错误。
func ErrDispatchRejected(reason string) *DispatchRejected {
	return &DispatchRejected{Reason: reason}
}
