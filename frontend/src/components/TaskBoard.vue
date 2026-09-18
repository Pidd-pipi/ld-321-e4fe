<script setup lang="ts">
import { reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { STATUS_COLORS, TASK_STATUS_PENDING } from '../constants/app.constants';
import { createDispatchOrder } from '../services/dispatch.service';
import type { Driver, FarmTask, Machine } from '../types/domain';

const props = defineProps<{ tasks: FarmTask[]; machines: Machine[]; drivers: Driver[] }>();
const emit = defineEmits<{ changed: [] }>();

// 每个任务的派单表单：默认带出系统推荐的农机与驾驶员。
const selections = reactive<Record<string, { machineId: string; driverId: string }>>({});
const submittingId = ref('');

const selectionOf = (task: FarmTask) => {
  if (!selections[task.id]) {
    const machine = props.machines.find((m) => m.code === task.recommendedMachine);
    const driver = props.drivers.find((d) => d.name === task.recommendedDriver);
    selections[task.id] = { machineId: machine?.id ?? '', driverId: driver?.id ?? '' };
  }
  return selections[task.id];
};

const handleDispatch = async (task: FarmTask) => {
  const selection = selectionOf(task);
  if (!selection.machineId || !selection.driverId) {
    ElMessage.warning('请选择农机与驾驶员');
    return;
  }
  submittingId.value = task.id;
  try {
    const result = await createDispatchOrder({
      taskId: task.id,
      machineId: selection.machineId,
      driverId: selection.driverId,
    });
    ElMessage.success(result.message);
    emit('changed');
  } catch (err) {
    // 整单拒绝：后端返回具体失败原因（农机占用/保养不足/驾驶员不在岗）。
    ElMessage.error(err instanceof Error ? err.message : '派单失败');
    emit('changed');
  } finally {
    submittingId.value = '';
  }
};
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">作业任务调度</h2>
    <div class="grid gap-3">
      <article v-for="task in tasks" :key="task.id" class="rounded-md border border-slate-200 p-3">
        <div class="flex items-start justify-between gap-3">
          <div>
            <div class="flex items-center gap-2">
              <strong>{{ task.type }} · {{ task.field }}</strong>
              <el-tag :type="STATUS_COLORS[task.status]" size="small">{{ task.status }}</el-tag>
            </div>
            <p class="mt-1 text-sm text-slate-600">
              {{ task.areaMu }} 亩 · 预计 {{ task.estimatedHours }} 小时 · {{ task.plannedWindow }}
            </p>
            <p class="mt-1 text-sm text-emerald-700">
              推荐 {{ task.recommendedMachine }} / {{ task.recommendedDriver }}
            </p>
          </div>
        </div>
        <div v-if="task.status === TASK_STATUS_PENDING" class="mt-3 flex flex-wrap items-center gap-2">
          <el-select
            v-model="selectionOf(task).machineId"
            placeholder="选择农机"
            size="small"
            class="w-52"
          >
            <el-option
              v-for="machine in machines"
              :key="machine.id"
              :label="`${machine.code}（${machine.status}）`"
              :value="machine.id"
            />
          </el-select>
          <el-select
            v-model="selectionOf(task).driverId"
            placeholder="选择驾驶员"
            size="small"
            class="w-44"
          >
            <el-option
              v-for="driver in drivers"
              :key="driver.id"
              :label="`${driver.name}（${driver.status}）`"
              :value="driver.id"
            />
          </el-select>
          <el-button
            size="small"
            type="primary"
            :loading="submittingId === task.id"
            @click="handleDispatch(task)"
          >提交派单</el-button>
        </div>
      </article>
    </div>
  </section>
</template>
