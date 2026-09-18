package util

import (
	"errors"
	"net/http"

	"github.com/agridispatch/agridispatch/internal/constants"
	apperrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/gin-gonic/gin"
)

// Response 统一响应体。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: constants.CodeOK, Message: "ok", Data: data})
}

// Created 创建成功响应。
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Code: constants.CodeOK, Message: "ok", Data: data})
}

// Fail 业务失败响应。
func Fail(c *gin.Context, status, code int, message string) {
	c.JSON(status, Response{Code: code, Message: message})
}

// FailError 根据错误类型转换为统一响应。
func FailError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		Fail(c, http.StatusNotFound, constants.CodeNotFound, err.Error())
	default:
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
	}
}

// FailDispatchError 派单闭环错误转换：业务拒绝/状态冲突返回 409 且携带可展示原因。
func FailDispatchError(c *gin.Context, err error) {
	var rejected *apperrors.DispatchRejectedError
	if errors.As(err, &rejected) {
		Fail(c, http.StatusConflict, constants.CodeConflict, rejected.Reason)
		return
	}
	var state *apperrors.DispatchStateError
	if errors.As(err, &state) {
		Fail(c, http.StatusConflict, constants.CodeConflict, state.Reason)
		return
	}
	FailError(c, err)
}
