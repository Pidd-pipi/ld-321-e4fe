export interface DashboardItem {
  id: string;
  title: string;
  description: string;
  status: string;
  score: number;
}

export interface Machine {
  id: string;
  code: string;
  name: string;
  model: string;
  purchasedAt: string;
  horsepower: number;
  field: string;
  status: string;
  qrCode: string;
  photoUrl: string;
  workHours: number;
  currentTask: string;
  /** 距下次保养剩余可作业工时 */
  maintenanceHours: number;
  /** 当前绑定的任务 ID（作业中时有值） */
  taskId: string;
}

export interface FarmTask {
  id: string;
  type: string;
  field: string;
  areaMu: number;
  estimatedHours: number;
  status: string;
  priority: string;
  recommendedMachine: string;
  recommendedDriver: string;
  plannedWindow: string;
  /** 最近一次派单/取消尝试（刷新后仍可回读失败原因） */
  lastAttemptAction: string;
  lastAttemptResult: string;
  lastAttemptReason: string;
  lastAttemptAt: string;
}

/** 派单/取消尝试记录 */
export interface DispatchAttempt {
  id: number;
  taskId: string;
  taskType: string;
  taskField: string;
  machineCode: string;
  driverName: string;
  action: 'dispatch' | 'cancel';
  result: 'success' | 'rejected' | 'released' | 'noop';
  reason: string;
  createdAt: string;
}

export interface TrackPoint {
  machineCode: string;
  taskType: string;
  capturedAt: string;
  longitude: number;
  latitude: number;
  speed: number;
  fieldBoundary: string;
}

export interface WorkRecord {
  id: string;
  machineCode: string;
  driverName: string;
  workDate: string;
  taskType: string;
  actualHours: number;
  fuelLiters: number;
  areaMu: number;
  fuelCost: number;
}

export interface MaintenanceReminder {
  id: string;
  machineCode: string;
  title: string;
  dueDate: string;
  remainingHours: number;
  level: string;
  lastServiceRecord: string;
}

export interface Driver {
  id: string;
  name: string;
  licenseNo: string;
  phone: string;
  shift: string;
  restDay: string;
  monthAreaMu: number;
  rating: number;
  /** 当天在岗状态：在岗/休息 */
  status: string;
  /** 作业占用状态：空闲/作业中 */
  workStatus: string;
  taskId: string;
}

export interface DispatchBoard {
  todayTodos: number;
  idleMachines: number;
  workingMachines: string[];
  dueMaintenance: string[];
  sevenDayAreas: number[];
  trendLabels: string[];
}

export interface FarmOverview {
  items: DashboardItem[];
  machines: Machine[];
  tasks: FarmTask[];
  tracks: TrackPoint[];
  records: WorkRecord[];
  maintenance: MaintenanceReminder[];
  drivers: Driver[];
  attempts: DispatchAttempt[];
  board: DispatchBoard;
  stats: {
    totalAreaMu: number;
    totalHours: number;
    fuelCost: number;
  };
}
