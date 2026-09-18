package constants

// 派单领域使用的状态、动作与结果常量，禁止在业务代码中硬编码。

// 农机状态
const (
	MachineIdle    = "空闲"
	MachineWorking = "作业中"
	MachineRepair  = "维修中"
)

// 任务状态
const (
	TaskPending    = "待派单"
	TaskDispatched = "作业中"
	TaskDone       = "已完成"
	TaskCancelled  = "已取消"
)

// 驾驶员当日在岗状态（Status：当天能否派班）
const (
	DriverOnDuty  = "在岗"
	DriverOffDuty = "休息"
)

// 驾驶员作业占用状态（WorkStatus：当前是否被任务占用）
const (
	DriverIdle    = "空闲"
	DriverWorking = "作业中"
)

// 派单记录动作
const (
	DispatchActionDispatch = "dispatch"
	DispatchActionCancel   = "cancel"
)

// 派单记录结果
const (
	DispatchResultSuccess = "success"
	DispatchResultReject  = "rejected"
	DispatchResultRelease = "released"
	DispatchResultNoop    = "noop"
)

// 派单被拒原因（同时作为面向用户的中文提示）
const (
	ReasonTaskNotPending     = "任务已派单或已结束，无需重复派单"
	ReasonTaskCancelled      = "任务已取消，不能派单"
	ReasonTaskMissingBinding = "任务未选择农机或驾驶员，无法派单"
	ReasonMachineNotFound    = "指定的农机不存在"
	ReasonDriverNotFound     = "指定的驾驶员不存在"
	ReasonMachineNotIdle     = "农机当前非空闲状态（可能已被其他任务占用）"
	ReasonMaintenanceShort   = "农机保养剩余工时不足，预计作业时长将超过保养间隔"
	ReasonDriverOffDuty      = "驾驶员当天不在岗（休息或已被排班为不可用）"
	ReasonDriverBusy         = "驾驶员已在执行其他任务"
	ReasonNotDispatched      = "任务当前不是作业中状态，无可释放的派单占用"
)

// 保养工时比较允许的浮点误差
const MaintenanceHoursEpsilon = 1e-9
