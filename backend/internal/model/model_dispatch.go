package model

import "time"

// DispatchAttempt 派单/取消尝试记录（每次提交都落库留痕，供看板回读结果与失败原因）。
type DispatchAttempt struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID      string    `gorm:"size:32;index" json:"taskId"`
	TaskType    string    `gorm:"size:32" json:"taskType"`
	TaskField   string    `gorm:"size:64" json:"taskField"`
	MachineCode string    `gorm:"size:32;index" json:"machineCode"`
	DriverName  string    `gorm:"size:64" json:"driverName"`
	Action      string    `gorm:"size:16;index" json:"action"` // dispatch / cancel
	Result      string    `gorm:"size:16;index" json:"result"` // success / rejected / released / noop
	Reason      string    `gorm:"size:255" json:"reason"`      // 失败/拒绝原因，成功时为空
	CreatedAt   time.Time `json:"createdAt"`
}
