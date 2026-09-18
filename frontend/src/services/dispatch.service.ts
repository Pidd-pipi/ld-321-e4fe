import { API_BASE } from '../constants/app.constants';
import { AppException } from '../errors/AppException';
import { logger } from '../logger/logger';
import type { DispatchOrder, DispatchOrderResult } from '../types/domain';

// 解析后端统一响应 {code, message, data}，失败时抛出带后端原因的异常。
const unwrap = async <T>(response: Response, fallback: string): Promise<T> => {
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const message = body && typeof body.message === 'string' ? body.message : fallback;
    logger.error('dispatch request failed', response.status, message);
    throw new AppException('DISPATCH_FAILED', message);
  }
  if (body && typeof body === 'object' && body.code === 0) {
    return body.data as T;
  }
  return body as T;
};

export const fetchDispatchOrders = async (): Promise<DispatchOrder[]> => {
  const response = await fetch(`${API_BASE}/dispatch/orders`);
  return unwrap<DispatchOrder[]>(response, '无法加载派单看板数据');
};

export const createDispatchOrder = async (payload: {
  taskId: string;
  machineId: string;
  driverId: string;
}): Promise<DispatchOrderResult> => {
  const response = await fetch(`${API_BASE}/dispatch/orders`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return unwrap<DispatchOrderResult>(response, '派单失败');
};

export const cancelDispatchOrder = async (orderId: string): Promise<DispatchOrderResult> => {
  const response = await fetch(`${API_BASE}/dispatch/orders/${orderId}/cancel`, { method: 'POST' });
  return unwrap<DispatchOrderResult>(response, '取消派单失败');
};
