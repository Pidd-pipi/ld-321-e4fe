package errors

import "fmt"

// BusinessError 业务错误。
type BusinessError struct {
	Code    int
	Message string
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("code=%d message=%s", e.Code, e.Message)
}

// New 构造业务错误。
func New(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

// ValidationError 参数校验错误。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation: " + e.Message
}

// MachineOfflineError 农机离线错误。
type MachineOfflineError struct {
	MachineCode string
}

func (e *MachineOfflineError) Error() string {
	return fmt.Sprintf("machine %s is offline", e.MachineCode)
}

// DispatchRejectedError 派单预检未通过（农机/保养/驾驶员任一条件不满足，整单拒绝）。
type DispatchRejectedError struct {
	Reason string
}

func (e *DispatchRejectedError) Error() string {
	return e.Reason
}

// DispatchStateError 派单状态冲突（重复提交或并发竞争，只有一方能成功）。
type DispatchStateError struct {
	Reason string
}

func (e *DispatchStateError) Error() string {
	return e.Reason
}
