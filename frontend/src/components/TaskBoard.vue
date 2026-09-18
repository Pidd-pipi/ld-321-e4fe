<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { STATUS_COLORS } from '../constants/app.constants';
import { dispatchTask, cancelDispatch } from '../services/storage.service';
import type { FarmTask } from '../types/domain';

const props = defineProps<{ tasks: FarmTask[] }>();
const emit = defineEmits<{ (e: 'changed'): void }>();

const pendingId = ref('');

const handleDispatch = async (task: FarmTask) => {
  pendingId.value = task.id;
  try {
    const result = await dispatchTask(task.id);
    ElMessage.success(result.message ?? '派单成功');
    emit('changed');
  } catch (err) {
    // 失败原因来自后端整单拒绝（409），直接展示
    ElMessage.error(err instanceof Error ? err.message : '派单失败');
    emit('changed');
  } finally {
    pendingId.value = '';
  }
};

const handleCancel = async (task: FarmTask) => {
  try {
    await ElMessageBox.confirm(
      `确定取消「${task.type} · ${task.field}」的派单？将同步释放农机与驾驶员。`,
      '取消派单',
      { type: 'warning', confirmButtonText: '取消派单', cancelButtonText: '再想想' },
    );
  } catch {
    return;
  }
  pendingId.value = task.id;
  try {
    const result = await cancelDispatch(task.id);
    if (result.released) {
      ElMessage.success(result.message);
    } else {
      ElMessage.info(result.message);
    }
    emit('changed');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '取消派单失败');
  } finally {
    pendingId.value = '';
  }
};
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">作业任务调度（派单预占闭环）</h2>
    <div class="grid gap-3">
      <article
        v-for="task in props.tasks"
        :key="task.id"
        class="rounded-md border"
        :class="task.lastAttemptResult === 'rejected' ? 'border-red-300 bg-red-50/60' : 'border-slate-200'"
      >
        <div class="flex items-start justify-between gap-3 p-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <strong>{{ task.type }} · {{ task.field }}</strong>
              <el-tag :type="STATUS_COLORS[task.status] ?? 'info'" size="small">{{ task.status }}</el-tag>
            </div>
            <p class="mt-1 text-sm text-slate-600">
              {{ task.areaMu }} 亩 · 预计 {{ task.estimatedHours }} 小时 · {{ task.plannedWindow }}
            </p>
            <p class="mt-1 text-sm" :class="task.status === '作业中' ? 'text-emerald-700 font-semibold' : 'text-slate-500'">
              绑定 {{ task.recommendedMachine }} / {{ task.recommendedDriver }}
            </p>

            <!-- 最近一次尝试结果：成功提示绑定关系，失败展示原因（刷新后仍回读） -->
            <p v-if="task.lastAttemptResult === 'rejected'" class="mt-2 text-sm text-red-600">
              <el-tag type="danger" size="small" effect="dark" class="mr-1">已拒绝</el-tag>
              {{ task.lastAttemptReason }}
            </p>
            <p v-else-if="task.lastAttemptResult === 'success' && task.status === '作业中'" class="mt-2 text-sm text-emerald-700">
              <el-tag type="success" size="small" effect="dark" class="mr-1">作业中</el-tag>
              农机与驾驶员已一起占用
            </p>
            <p v-else-if="task.lastAttemptResult === 'released'" class="mt-2 text-sm text-slate-600">
              <el-tag type="info" size="small" effect="dark" class="mr-1">已取消</el-tag>
              任务、农机与驾驶员占用已同步释放
            </p>
          </div>

          <div class="flex shrink-0 flex-col gap-2">
            <el-button
              v-if="task.status === '待派单'"
              size="small"
              type="primary"
              :loading="pendingId === task.id"
              @click="handleDispatch(task)"
            >
              一键派单
            </el-button>
            <el-button
              v-if="task.status === '作业中'"
              size="small"
              type="warning"
              plain
              :loading="pendingId === task.id"
              @click="handleCancel(task)"
            >
              取消派单
            </el-button>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
