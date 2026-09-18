package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	bizerrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/google/uuid"
)

// dispatchOrderListLimit 看板回读的派单记录上限。
const dispatchOrderListLimit = 100

// DispatchService 派单预占闭环服务。
type DispatchService struct {
	repo   *repository.DispatchRepository
	logger *slog.Logger
}

func NewDispatchService(repo *repository.DispatchRepository, logger *slog.Logger) *DispatchService {
	return &DispatchService{repo: repo, logger: logger}
}

// DispatchCheckInput 派单预检输入。
type DispatchCheckInput struct {
	Task           *model.FarmTask
	Machine        *model.Machine
	RemainingHours float64
	HasReminder    bool
	Driver         *model.Driver
	Weekday        string
}

// EvaluateDispatchPreconditions 派单预检：农机空闲、保养剩余小时充足、驾驶员当天在岗。
// 返回空串表示全部通过，否则返回失败原因（整单拒绝）。
func EvaluateDispatchPreconditions(in DispatchCheckInput) string {
	if in.Task.Status != constants.TaskPending {
		return fmt.Sprintf(constants.ReasonTaskNotPending, in.Task.Status)
	}
	if in.Machine == nil {
		return constants.ReasonMachineNotFound
	}
	if in.Machine.Status != constants.MachineIdle {
		return fmt.Sprintf(constants.ReasonMachineNotIdle, in.Machine.Code, in.Machine.Status)
	}
	if in.HasReminder && in.RemainingHours < in.Task.EstimatedHours {
		return fmt.Sprintf(constants.ReasonMaintenanceShort, in.Machine.Code, in.RemainingHours, in.Task.EstimatedHours)
	}
	if in.Driver == nil {
		return constants.ReasonDriverNotFound
	}
	if in.Driver.RestDay == in.Weekday {
		return fmt.Sprintf(constants.ReasonDriverRestDay, in.Driver.Name, in.Weekday)
	}
	if in.Driver.Status != constants.DriverOnDuty && in.Driver.Status != constants.DriverAvailable {
		return fmt.Sprintf(constants.ReasonDriverOffDuty, in.Driver.Name, in.Driver.Status)
	}
	return ""
}

// WeekdayCN 返回中文星期名。
func WeekdayCN(t time.Time) string {
	return constants.WeekdayNamesCN[int(t.Weekday())]
}

// Dispatch 提交派单：预检通过后在一个事务里把任务、农机、驾驶员一起标记为作业中；
// 任一条件不满足整单拒绝并持久化失败原因，不留下单向占用。
func (s *DispatchService) Dispatch(taskID, machineID, driverID string) (*model.DispatchOrder, error) {
	task, err := s.repo.FindTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	machine, machineErr := s.repo.FindMachineByID(machineID)
	if machineErr != nil && !errors.Is(machineErr, repository.ErrNotFound) {
		return nil, machineErr
	}
	driver, driverErr := s.repo.FindDriverByID(driverID)
	if driverErr != nil && !errors.Is(driverErr, repository.ErrNotFound) {
		return nil, driverErr
	}

	var remaining float64
	var hasReminder bool
	if machine != nil {
		remaining, hasReminder, err = s.repo.MinRemainingHours(machine.Code)
		if err != nil {
			return nil, err
		}
	}

	reason := EvaluateDispatchPreconditions(DispatchCheckInput{
		Task:           task,
		Machine:        machine,
		RemainingHours: remaining,
		HasReminder:    hasReminder,
		Driver:         driver,
		Weekday:        WeekdayCN(time.Now()),
	})
	if reason != "" {
		rejected := s.buildOrder(task, machine, driver, constants.OrderRejected)
		rejected.FailReason = reason
		if recErr := s.repo.RecordOrder(rejected); recErr != nil {
			s.logger.Error("record rejected dispatch failed", "err", recErr, "taskId", taskID)
		}
		s.logger.Info("dispatch rejected", "taskId", taskID, "reason", reason)
		return nil, &bizerrors.DispatchRejectedError{Reason: reason}
	}

	order := s.buildOrder(task, machine, driver, constants.OrderActive)
	currentTask := fmt.Sprintf("%s %s", task.Type, task.Field)
	if err := s.repo.CreateOrderTx(order, currentTask); err != nil {
		if errors.Is(err, repository.ErrTaskStateChanged) ||
			errors.Is(err, repository.ErrMachineStateChanged) ||
			errors.Is(err, repository.ErrDriverStateChanged) {
			return nil, &bizerrors.DispatchStateError{Reason: constants.ReasonResourceConflict}
		}
		return nil, err
	}
	s.logger.Info("dispatch occupied",
		"orderId", order.ID, "taskId", taskID,
		"machine", order.MachineCode, "driver", order.DriverName)
	return order, nil
}

// Cancel 取消派单：同事务释放任务、农机、驾驶员；重复或并发取消只能成功一次。
func (s *DispatchService) Cancel(orderID string) (*model.DispatchOrder, error) {
	order, err := s.repo.CancelOrderTx(orderID, time.Now())
	if err != nil {
		if errors.Is(err, repository.ErrOrderAlreadySettled) {
			return nil, &bizerrors.DispatchStateError{Reason: constants.ReasonOrderAlreadyClose}
		}
		return nil, err
	}
	s.logger.Info("dispatch released",
		"orderId", order.ID, "taskId", order.TaskID,
		"machine", order.MachineCode, "driver", order.DriverName)
	return order, nil
}

// ListOrders 看板回读派单绑定关系与失败原因。
func (s *DispatchService) ListOrders() ([]model.DispatchOrder, error) {
	return s.repo.ListOrders(dispatchOrderListLimit)
}

// buildOrder 组装派单记录（冗余任务/农机/驾驶员快照，绑定关系刷新后仍可回读）。
func (s *DispatchService) buildOrder(task *model.FarmTask, machine *model.Machine, driver *model.Driver, status string) *model.DispatchOrder {
	order := &model.DispatchOrder{
		ID:        "do-" + uuid.NewString(),
		TaskID:    task.ID,
		TaskType:  task.Type,
		TaskField: task.Field,
		Status:    status,
	}
	if machine != nil {
		order.MachineID = machine.ID
		order.MachineCode = machine.Code
	}
	if driver != nil {
		order.DriverID = driver.ID
		order.DriverName = driver.Name
	}
	return order
}
