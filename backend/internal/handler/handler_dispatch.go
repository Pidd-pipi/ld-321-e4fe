package handler

import (
	"net/http"

	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/util"
	"github.com/gin-gonic/gin"
)

// DispatchHandler 派单预占闭环处理器。
type DispatchHandler struct {
	dispatchSvc *service.DispatchService
}

func NewDispatchHandler(dispatchSvc *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{dispatchSvc: dispatchSvc}
}

// CreateOrderRequest 提交派单请求。
type CreateOrderRequest struct {
	TaskID    string `json:"taskId" binding:"required"`
	MachineID string `json:"machineId" binding:"required"`
	DriverID  string `json:"driverId" binding:"required"`
}

// CreateOrder 提交派单（预占任务/农机/驾驶员，任一条件不满足整单拒绝）。
func (h *DispatchHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, 40000, "taskId、machineId、driverId 均为必填")
		return
	}
	order, err := h.dispatchSvc.Dispatch(req.TaskID, req.MachineID, req.DriverID)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, gin.H{"order": order, "message": "派单成功，任务/农机/驾驶员已同步标记为作业中"})
}

// CancelOrder 取消派单（同步释放任务、农机、驾驶员，仅可成功一次）。
func (h *DispatchHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		util.Fail(c, http.StatusBadRequest, 40000, "order id required")
		return
	}
	order, err := h.dispatchSvc.Cancel(orderID)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, gin.H{"order": order, "message": "派单已取消，任务/农机/驾驶员已同步释放"})
}

// ListOrders 看板回读派单绑定关系与失败原因。
func (h *DispatchHandler) ListOrders(c *gin.Context) {
	orders, err := h.dispatchSvc.ListOrders()
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, orders)
}
