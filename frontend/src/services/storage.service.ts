import { API_BASE } from '../constants/app.constants';
import { AppException } from '../errors/AppException';
import { logger } from '../logger/logger';
import type { DashboardItem, FarmOverview } from '../types/domain';

export const fetchFarmOverview = async (): Promise<FarmOverview> => {
  const response = await fetch(`${API_BASE}/dashboard/overview`);
  if (!response.ok) {
    logger.error('overview request failed', response.status);
    throw new AppException('OVERVIEW_FAILED', '无法加载农机调度看板数据');
  }
  const body = await response.json();
  // 解包后端统一响应 {code, message, data}
  if (body && typeof body === 'object' && body.code === 0 && body.data) {
    return body.data as FarmOverview;
  }
  return body as FarmOverview;
};

export const dispatchTask = async (taskId: string) => {
  const response = await fetch(`${API_BASE}/tasks/${taskId}/dispatch`, { method: 'POST' });
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    // 409：条件不满足被整单拒绝，message 为可直接展示的失败原因
    throw new AppException('DISPATCH_REJECTED', body?.message || '派单失败：条件不满足，已整单拒绝');
  }
  // 解包后端统一响应 {code, message, data}
  if (body && typeof body === 'object' && body.code === 0 && body.data) {
    return body.data;
  }
  return body;
};

export const cancelDispatch = async (taskId: string) => {
  const response = await fetch(`${API_BASE}/tasks/${taskId}/cancel`, { method: 'POST' });
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    throw new AppException('CANCEL_FAILED', body?.message || '取消派单失败');
  }
  if (body && typeof body === 'object' && body.code === 0 && body.data) {
    return body.data as { released: boolean; message: string };
  }
  return body;
};

export const saveItems = (items: DashboardItem[]) => {
  localStorage.setItem('agridispatch.items', JSON.stringify(items));
};
