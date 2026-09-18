package service

import (
	"testing"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
)

func pendingTask() *model.FarmTask {
	return &model.FarmTask{ID: "t1", Type: "播种", Field: "西坡旱地", Status: constants.TaskPending, EstimatedHours: 6}
}

func idleMachine() *model.Machine {
	return &model.Machine{ID: "m1", Code: "NJ-2026-002", Status: constants.MachineIdle}
}

func onDutyDriver() *model.Driver {
	return &model.Driver{ID: "d1", Name: "何燕", Status: constants.DriverOnDuty, RestDay: "周三"}
}

func TestEvaluateDispatchPreconditions(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(in *DispatchCheckInput)
		wantReason bool
	}{
		{
			name:       "全部条件满足",
			mutate:     func(in *DispatchCheckInput) {},
			wantReason: false,
		},
		{
			name:       "任务非待派单",
			mutate:     func(in *DispatchCheckInput) { in.Task.Status = constants.TaskWorking },
			wantReason: true,
		},
		{
			name:       "农机不存在",
			mutate:     func(in *DispatchCheckInput) { in.Machine = nil },
			wantReason: true,
		},
		{
			name:       "农机作业中",
			mutate:     func(in *DispatchCheckInput) { in.Machine.Status = constants.MachineWorking },
			wantReason: true,
		},
		{
			name:       "农机维修中",
			mutate:     func(in *DispatchCheckInput) { in.Machine.Status = constants.MachineRepair },
			wantReason: true,
		},
		{
			name: "保养剩余小时不足",
			mutate: func(in *DispatchCheckInput) {
				in.HasReminder = true
				in.RemainingHours = 5.5
			},
			wantReason: true,
		},
		{
			name: "保养剩余小时恰好等于预计时长",
			mutate: func(in *DispatchCheckInput) {
				in.HasReminder = true
				in.RemainingHours = 6
			},
			wantReason: false,
		},
		{
			name: "无保养提醒不校验工时",
			mutate: func(in *DispatchCheckInput) {
				in.HasReminder = false
				in.RemainingHours = 0
			},
			wantReason: false,
		},
		{
			name:       "驾驶员不存在",
			mutate:     func(in *DispatchCheckInput) { in.Driver = nil },
			wantReason: true,
		},
		{
			name:       "驾驶员当天休息",
			mutate:     func(in *DispatchCheckInput) { in.Driver.RestDay = "周五" },
			wantReason: true,
		},
		{
			name:       "驾驶员状态休息",
			mutate:     func(in *DispatchCheckInput) { in.Driver.Status = constants.DriverRest },
			wantReason: true,
		},
		{
			name:       "驾驶员已被占用",
			mutate:     func(in *DispatchCheckInput) { in.Driver.Status = constants.DriverWorking },
			wantReason: true,
		},
		{
			name:       "驾驶员可派单也算在岗",
			mutate:     func(in *DispatchCheckInput) { in.Driver.Status = constants.DriverAvailable },
			wantReason: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := DispatchCheckInput{
				Task:           pendingTask(),
				Machine:        idleMachine(),
				RemainingHours: 42,
				HasReminder:    true,
				Driver:         onDutyDriver(),
				Weekday:        "周五",
			}
			tt.mutate(&in)
			reason := EvaluateDispatchPreconditions(in)
			if tt.wantReason && reason == "" {
				t.Errorf("expected rejection reason, got pass")
			}
			if !tt.wantReason && reason != "" {
				t.Errorf("expected pass, got reason %q", reason)
			}
		})
	}
}

func TestWeekdayCN(t *testing.T) {
	// 2026-09-18 是周五。
	d := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	if got := WeekdayCN(d); got != "周五" {
		t.Errorf("WeekdayCN(2026-09-18) = %q, want 周五", got)
	}
	// 2026-09-20 是周日。
	d = time.Date(2026, 9, 20, 12, 0, 0, 0, time.Local)
	if got := WeekdayCN(d); got != "周日" {
		t.Errorf("WeekdayCN(2026-09-20) = %q, want 周日", got)
	}
}
