package errors

import "fmt"

// DispatchRejectedError 派单预占被业务规则拒绝（条件不满足，整单未占用）。
// 属于可预期的业务结果，HTTP 层映射为 409 Conflict，Reason 直接展示在看板上。
type DispatchRejectedError struct {
	Reason string
}

func (e *DispatchRejectedError) Error() string {
	return fmt.Sprintf("dispatch rejected: %s", e.Reason)
}

// NewDispatchRejected 构造派单拒绝错误。
func NewDispatchRejected(reason string) *DispatchRejectedError {
	return &DispatchRejectedError{Reason: reason}
}

// DispatchStateError 派单/取消时任务状态与预期不符（重复操作、无可释放占用等）。
type DispatchStateError struct {
	Reason string
}

func (e *DispatchStateError) Error() string {
	return fmt.Sprintf("dispatch state conflict: %s", e.Reason)
}

// NewDispatchState 构造派单状态冲突错误。
func NewDispatchState(reason string) *DispatchStateError {
	return &DispatchStateError{Reason: reason}
}
