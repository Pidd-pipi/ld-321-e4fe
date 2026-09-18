package model

import "time"

// DispatchOrder 派单预占记录：持久化任务-农机-驾驶员绑定关系与失败原因。
type DispatchOrder struct {
	ID          string     `gorm:"primaryKey;size:40" json:"id"`
	TaskID      string     `gorm:"size:32;index" json:"taskId"`
	TaskType    string     `gorm:"size:32" json:"taskType"`
	TaskField   string     `gorm:"size:64" json:"taskField"`
	MachineID   string     `gorm:"size:32;index" json:"machineId"`
	MachineCode string     `gorm:"size:32" json:"machineCode"`
	DriverID    string     `gorm:"size:32;index" json:"driverId"`
	DriverName  string     `gorm:"size:64" json:"driverName"`
	Status      string     `gorm:"size:16;index" json:"status"`
	FailReason  string     `gorm:"size:255" json:"failReason"`
	CancelledAt *time.Time `json:"cancelledAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"-"`
}
