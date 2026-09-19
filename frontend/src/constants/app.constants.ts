export const APP_NAME = 'AgriDispatch 农机调度';
export const API_BASE = '/api';
export const STATUS_COLORS: Record<string, string> = {
  空闲: 'success',
  作业中: 'warning',
  维修中: 'danger',
  待派单: 'info',
  已派单: 'warning',
  已完成: 'success',
  已取消: 'info',
  在岗: 'success',
  休息: 'danger',
  可派单: 'success',
};

// 派单尝试结果标签配色
export const ATTEMPT_RESULT_COLORS: Record<string, string> = {
  success: 'success',
  rejected: 'danger',
  released: 'info',
  noop: 'info',
};

export const ATTEMPT_RESULT_TEXT: Record<string, string> = {
  success: '派单成功',
  rejected: '整单拒绝',
  released: '已释放',
  noop: '重复取消',
};

export const ATTEMPT_ACTION_TEXT: Record<string, string> = {
  dispatch: '派单',
  cancel: '取消',
};
