package constants

// 应用常量。
const (
	AppName                 = "agridispatch"
	APIVersion              = "v1"
	PageSizeDefault         = 10
	PageSizeMax             = 100
	OverviewCacheKey        = "agridispatch:overview"
	OverviewCacheTTLSeconds = 30
	WSPushIntervalSeconds   = 5
)

// 角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 农机状态
const (
	MachineIdle    = "空闲"
	MachineWorking = "作业中"
	MachineRepair  = "维修中"
)

// 任务状态
const (
	TaskPending    = "待派单"
	TaskDispatched = "已派单"
	TaskWorking    = "作业中"
	TaskDone       = "已完成"
)

// 驾驶员状态
const (
	DriverOnDuty    = "在岗"
	DriverAvailable = "可派单"
	DriverWorking   = "作业中"
	DriverRest      = "休息"
)

// 派单状态
const (
	OrderActive    = "生效中"
	OrderCancelled = "已取消"
	OrderRejected  = "已拒绝"
)

// 农机空闲时的在办任务占位文案
const MachineIdleTask = "可派单"

// 星期中文名（下标与 time.Weekday 一致，0=周日）。
var WeekdayNamesCN = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

// 派单失败原因模板
const (
	ReasonTaskNotPending    = "任务当前状态为「%s」，不允许派单"
	ReasonMachineNotFound   = "所选农机不存在"
	ReasonMachineNotIdle    = "农机 %s 当前状态为「%s」，不是空闲状态"
	ReasonMaintenanceShort  = "农机 %s 保养剩余 %.1f 小时，不足任务预计 %.1f 小时"
	ReasonDriverNotFound    = "所选驾驶员不存在"
	ReasonDriverRestDay     = "驾驶员 %s 今日（%s）休息，不在岗"
	ReasonDriverOffDuty     = "驾驶员 %s 当前状态为「%s」，不在岗"
	ReasonResourceConflict  = "资源状态已变化，请刷新看板后重试"
	ReasonOrderAlreadyClose = "该派单已取消，请勿重复操作"
)

// 错误码
const (
	CodeOK              = 0
	CodeBadRequest      = 40000
	CodeUnauthorized    = 40100
	CodeForbidden       = 40300
	CodeNotFound        = 40400
	CodeConflict        = 40900
	CodeInternalError   = 50000
	CodeTooManyRequests = 42900
)
