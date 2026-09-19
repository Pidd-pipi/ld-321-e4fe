-- 农机调度管理系统 数据库初始化脚本 (MySQL 8.0)
CREATE DATABASE IF NOT EXISTS agridispatch DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE agridispatch;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(20) NOT NULL,
  real_name VARCHAR(64) DEFAULT '',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS machines (
  id VARCHAR(32) PRIMARY KEY,
  code VARCHAR(32) NOT NULL UNIQUE,
  name VARCHAR(64) DEFAULT '',
  model VARCHAR(64) DEFAULT '',
  purchased_at VARCHAR(32) DEFAULT '',
  horsepower INT DEFAULT 0,
  field VARCHAR(64) DEFAULT '',
  status VARCHAR(20) DEFAULT '空闲',
  qr_code VARCHAR(64) DEFAULT '',
  photo_url VARCHAR(255) DEFAULT '',
  work_hours DECIMAL(10,2) DEFAULT 0,
  current_task VARCHAR(64) DEFAULT '',
  maintenance_hours DECIMAL(8,2) DEFAULT 0 COMMENT '距下次保养剩余可作业工时',
  task_id VARCHAR(32) DEFAULT '' COMMENT '当前绑定任务ID',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_machine_task (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS farm_tasks (
  id VARCHAR(32) PRIMARY KEY,
  type VARCHAR(32) DEFAULT '',
  field VARCHAR(64) DEFAULT '',
  area_mu DECIMAL(10,2) DEFAULT 0,
  estimated_hours DECIMAL(8,2) DEFAULT 0,
  status VARCHAR(20) DEFAULT '待派单',
  priority VARCHAR(16) DEFAULT '中',
  recommended_machine VARCHAR(64) DEFAULT '',
  recommended_driver VARCHAR(64) DEFAULT '',
  planned_window VARCHAR(64) DEFAULT '',
  last_attempt_action VARCHAR(16) DEFAULT '' COMMENT '最近尝试 dispatch/cancel',
  last_attempt_result VARCHAR(16) DEFAULT '' COMMENT 'success/rejected/released/noop',
  last_attempt_reason VARCHAR(255) DEFAULT '' COMMENT '最近尝试失败原因',
  last_attempt_at DATETIME(3) NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_task_last_attempt (last_attempt_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS track_points (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  machine_code VARCHAR(32) NOT NULL,
  task_type VARCHAR(32) DEFAULT '',
  captured_at VARCHAR(32) DEFAULT '',
  longitude DECIMAL(10,6) DEFAULT 0,
  latitude DECIMAL(10,6) DEFAULT 0,
  speed DECIMAL(8,2) DEFAULT 0,
  field_boundary VARCHAR(64) DEFAULT '',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_track_machine (machine_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS work_records (
  id VARCHAR(32) PRIMARY KEY,
  machine_code VARCHAR(32) NOT NULL,
  driver_name VARCHAR(64) DEFAULT '',
  work_date VARCHAR(32) DEFAULT '',
  task_type VARCHAR(32) DEFAULT '',
  actual_hours DECIMAL(8,2) DEFAULT 0,
  fuel_liters DECIMAL(8,2) DEFAULT 0,
  area_mu DECIMAL(10,2) DEFAULT 0,
  fuel_cost DECIMAL(10,2) DEFAULT 0,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_record_machine (machine_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS maintenance_reminders (
  id VARCHAR(32) PRIMARY KEY,
  machine_code VARCHAR(32) NOT NULL,
  title VARCHAR(128) DEFAULT '',
  due_date VARCHAR(32) DEFAULT '',
  remaining_hours DECIMAL(8,2) DEFAULT 0,
  level VARCHAR(16) DEFAULT 'normal',
  last_service_record VARCHAR(128) DEFAULT '',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS drivers (
  id VARCHAR(32) PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  license_no VARCHAR(32) DEFAULT '',
  phone VARCHAR(32) DEFAULT '',
  shift VARCHAR(16) DEFAULT '',
  rest_day VARCHAR(16) DEFAULT '',
  month_area_mu DECIMAL(10,2) DEFAULT 0,
  rating DECIMAL(4,2) DEFAULT 0,
  status VARCHAR(16) DEFAULT '在岗' COMMENT '当天在岗状态 在岗/休息',
  work_status VARCHAR(16) DEFAULT '空闲' COMMENT '作业占用状态 空闲/作业中',
  task_id VARCHAR(32) DEFAULT '' COMMENT '当前绑定任务ID',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_driver_task (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS dispatch_attempts (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  task_id VARCHAR(32) NOT NULL,
  task_type VARCHAR(32) DEFAULT '',
  task_field VARCHAR(64) DEFAULT '',
  machine_code VARCHAR(32) DEFAULT '',
  driver_name VARCHAR(64) DEFAULT '',
  action VARCHAR(16) NOT NULL COMMENT 'dispatch/cancel',
  result VARCHAR(16) NOT NULL COMMENT 'success/rejected/released/noop',
  reason VARCHAR(255) DEFAULT '' COMMENT '拒绝/失败原因',
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_attempt_task (task_id),
  INDEX idx_attempt_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS dashboard_items (
  id VARCHAR(32) PRIMARY KEY,
  title VARCHAR(64) DEFAULT '',
  description VARCHAR(128) DEFAULT '',
  status VARCHAR(16) DEFAULT '',
  score INT DEFAULT 0,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
