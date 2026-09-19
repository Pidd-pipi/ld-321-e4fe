package handler

import (
	"net/http"

	"github.com/agridispatch/agridispatch/internal/constants"
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

// Dispatch 提交派单（条件全部满足才原子占用，否则整单拒绝并回写原因）。
func (h *DispatchHandler) Dispatch(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "task id required")
		return
	}
	res, err := h.dispatchSvc.Dispatch(c.Request.Context(), taskID)
	if err != nil {
		util.FailDispatchError(c, err)
		return
	}
	util.OK(c, res)
}

// Cancel 取消派单（原子释放三方，重复/并发取消只生效一次）。
func (h *DispatchHandler) Cancel(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "task id required")
		return
	}
	res, err := h.dispatchSvc.Cancel(c.Request.Context(), taskID)
	if err != nil {
		util.FailDispatchError(c, err)
		return
	}
	util.OK(c, res)
}
