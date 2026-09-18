package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"gorm.io/gorm"
)

// 派单并发哨兵错误：条件更新命中 0 行说明资源状态已被并发请求改变。
var (
	ErrTaskStateChanged    = errors.New("task state changed")
	ErrMachineStateChanged = errors.New("machine state changed")
	ErrDriverStateChanged  = errors.New("driver state changed")
	ErrOrderAlreadySettled = errors.New("dispatch order already settled")
)

// DispatchRepository 派单数据访问。
type DispatchRepository struct {
	db *gorm.DB
}

func NewDispatchRepository(db *gorm.DB) *DispatchRepository {
	return &DispatchRepository{db: db}
}

// FindTaskByID 查找任务。
func (r *DispatchRepository) FindTaskByID(id string) (*model.FarmTask, error) {
	var t model.FarmTask
	err := r.db.First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	return &t, nil
}

// FindMachineByID 查找农机。
func (r *DispatchRepository) FindMachineByID(id string) (*model.Machine, error) {
	var m model.Machine
	err := r.db.First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find machine: %w", err)
	}
	return &m, nil
}

// FindDriverByID 查找驾驶员。
func (r *DispatchRepository) FindDriverByID(id string) (*model.Driver, error) {
	var d model.Driver
	err := r.db.First(&d, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find driver: %w", err)
	}
	return &d, nil
}

// MinRemainingHours 返回农机全部保养提醒中的最小剩余小时；无提醒时 hasReminder=false。
func (r *DispatchRepository) MinRemainingHours(machineCode string) (remaining float64, hasReminder bool, err error) {
	var reminders []model.MaintenanceReminder
	if err := r.db.Where("machine_code = ?", machineCode).Find(&reminders).Error; err != nil {
		return 0, false, fmt.Errorf("list maintenance reminders: %w", err)
	}
	if len(reminders) == 0 {
		return 0, false, nil
	}
	remaining = reminders[0].RemainingHours
	for _, rem := range reminders[1:] {
		if rem.RemainingHours < remaining {
			remaining = rem.RemainingHours
		}
	}
	return remaining, true, nil
}

// CreateOrderTx 在单事务内条件更新任务/农机/驾驶员并落派单，任一步失败整体回滚，不留单向占用。
func (r *DispatchRepository) CreateOrderTx(order *model.DispatchOrder, currentTask string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.FarmTask{}).
			Where("id = ? AND status = ?", order.TaskID, constants.TaskPending).
			Update("status", constants.TaskWorking)
		if res.Error != nil {
			return fmt.Errorf("lock task: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrTaskStateChanged
		}

		res = tx.Model(&model.Machine{}).
			Where("id = ? AND status = ?", order.MachineID, constants.MachineIdle).
			Updates(map[string]interface{}{
				"status":       constants.MachineWorking,
				"current_task": currentTask,
			})
		if res.Error != nil {
			return fmt.Errorf("lock machine: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrMachineStateChanged
		}

		res = tx.Model(&model.Driver{}).
			Where("id = ? AND status IN ?", order.DriverID, []string{constants.DriverOnDuty, constants.DriverAvailable}).
			Update("status", constants.DriverWorking)
		if res.Error != nil {
			return fmt.Errorf("lock driver: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrDriverStateChanged
		}

		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("insert dispatch order: %w", err)
		}
		return nil
	})
}

// CancelOrderTx 在单事务内关闭派单并同步释放任务、农机、驾驶员；重复取消命中 0 行直接失败。
func (r *DispatchRepository) CancelOrderTx(orderID string, now time.Time) (*model.DispatchOrder, error) {
	var order model.DispatchOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.DispatchOrder{}).
			Where("id = ? AND status = ?", orderID, constants.OrderActive).
			Updates(map[string]interface{}{
				"status":       constants.OrderCancelled,
				"cancelled_at": now,
			})
		if res.Error != nil {
			return fmt.Errorf("settle dispatch order: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			var existing model.DispatchOrder
			findErr := tx.First(&existing, "id = ?", orderID).Error
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			if findErr != nil {
				return fmt.Errorf("find dispatch order: %w", findErr)
			}
			return ErrOrderAlreadySettled
		}

		if err := tx.First(&order, "id = ?", orderID).Error; err != nil {
			return fmt.Errorf("reload dispatch order: %w", err)
		}
		if err := tx.Model(&model.FarmTask{}).
			Where("id = ?", order.TaskID).
			Update("status", constants.TaskPending).Error; err != nil {
			return fmt.Errorf("release task: %w", err)
		}
		if err := tx.Model(&model.Machine{}).
			Where("id = ?", order.MachineID).
			Updates(map[string]interface{}{
				"status":       constants.MachineIdle,
				"current_task": constants.MachineIdleTask,
			}).Error; err != nil {
			return fmt.Errorf("release machine: %w", err)
		}
		if err := tx.Model(&model.Driver{}).
			Where("id = ?", order.DriverID).
			Update("status", constants.DriverOnDuty).Error; err != nil {
			return fmt.Errorf("release driver: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// RecordOrder 落一条派单记录（用于持久化整单拒绝的失败原因）。
func (r *DispatchRepository) RecordOrder(order *model.DispatchOrder) error {
	if err := r.db.Create(order).Error; err != nil {
		return fmt.Errorf("record dispatch order: %w", err)
	}
	return nil
}

// ListOrders 按创建时间倒序返回派单记录（看板绑定关系与失败原因回读）。
func (r *DispatchRepository) ListOrders(limit int) ([]model.DispatchOrder, error) {
	var orders []model.DispatchOrder
	if err := r.db.Order("created_at DESC").Limit(limit).Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("list dispatch orders: %w", err)
	}
	return orders, nil
}
