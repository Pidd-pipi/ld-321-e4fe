package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/agridispatch/agridispatch/internal/constants"
	apperrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/redis/go-redis/v9"
)

// DispatchService 派单预占闭环服务：派单、取消，并维护看板缓存一致性。
type DispatchService struct {
	repo   *repository.DispatchRepository
	redis  *redis.Client
	logger *slog.Logger
}

func NewDispatchService(repo *repository.DispatchRepository, redis *redis.Client, logger *slog.Logger) *DispatchService {
	return &DispatchService{repo: repo, redis: redis, logger: logger}
}

// DispatchResult 派单结果（返回给前端/看板）。
type DispatchResult struct {
	TaskID      string `json:"taskId"`
	Status      string `json:"status"`
	MachineCode string `json:"machineCode"`
	DriverName  string `json:"driverName"`
	Message     string `json:"message"`
}

// Dispatch 提交派单：农机空闲、保养剩余工时不少于预计时长、驾驶员当天在岗，
// 三者同时满足才原子占用；否则整单拒绝（已由仓储落库失败原因）。
func (s *DispatchService) Dispatch(ctx context.Context, taskID string) (*DispatchResult, error) {
	if taskID == "" {
		return nil, apperrors.NewDispatchState(constants.ReasonTaskMissingBinding)
	}
	res, err := s.repo.Dispatch(taskID)
	if err != nil {
		var rejected *repository.DispatchRejected
		if errors.As(err, &rejected) {
			s.logger.Warn("dispatch rejected", "taskId", taskID, "reason", rejected.Reason)
			return nil, apperrors.NewDispatchRejected(rejected.Reason)
		}
		return nil, fmt.Errorf("dispatch task %s: %w", taskID, err)
	}
	s.invalidate(ctx)
	s.logger.Info("task dispatched atomically",
		"taskId", res.TaskID, "machine", res.MachineCode, "driver", res.DriverName)
	return &DispatchResult{
		TaskID:      res.TaskID,
		Status:      constants.TaskDispatched,
		MachineCode: res.MachineCode,
		DriverName:  res.DriverName,
		Message:     "派单成功：任务、农机与驾驶员已一起标记为作业中",
	}, nil
}

// CancelResult 取消结果。
type CancelResult struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	Released bool   `json:"released"`
	Message  string `json:"message"`
}

// Cancel 取消派单：原子释放任务、农机、驾驶员；重复/并发取消仅首次真正生效。
func (s *DispatchService) Cancel(ctx context.Context, taskID string) (*CancelResult, error) {
	if taskID == "" {
		return nil, apperrors.NewDispatchState(constants.ReasonTaskMissingBinding)
	}
	res, err := s.repo.Cancel(taskID)
	if err != nil {
		return nil, fmt.Errorf("cancel dispatch %s: %w", taskID, err)
	}
	s.invalidate(ctx)
	if !res.Released {
		s.logger.Warn("cancel ignored as idempotent noop", "taskId", taskID, "reason", res.Reason)
		return &CancelResult{
			TaskID: taskID, Status: constants.TaskPending, Released: false,
			Message: "该任务当前没有进行中的派单占用（重复取消，未重复释放）",
		}, nil
	}
	s.logger.Info("dispatch released atomically", "taskId", taskID)
	return &CancelResult{
		TaskID: taskID, Status: constants.TaskPending, Released: true,
		Message: "已取消派单：任务、农机与驾驶员占用已同步释放",
	}, nil
}

func (s *DispatchService) invalidate(ctx context.Context) {
	if err := s.redis.Del(ctx, constants.OverviewCacheKey).Err(); err != nil {
		s.logger.Warn("invalidate overview cache failed", "err", err)
	}
}
