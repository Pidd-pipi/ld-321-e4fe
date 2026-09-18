export const APP_NAME = 'AgriDispatch 农机调度';
export const API_BASE = '/api';
export const STATUS_COLORS: Record<string, string> = {
  空闲: 'success',
  作业中: 'warning',
  维修中: 'danger',
  待派单: 'info',
  已派单: 'warning',
  已完成: 'success',
};
export const ORDER_STATUS_COLORS: Record<string, string> = {
  生效中: 'success',
  已取消: 'info',
  已拒绝: 'danger',
};
export const TASK_STATUS_PENDING = '待派单';
export const ORDER_STATUS_ACTIVE = '生效中';
