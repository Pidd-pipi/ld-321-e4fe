<script setup lang="ts">
import { computed, ref } from 'vue';
import { STATUS_COLORS } from '../constants/app.constants';
import type { Machine } from '../types/domain';

const props = defineProps<{ machines: Machine[] }>();

const filter = ref('all');
const filtered = computed(() => {
  if (filter.value === 'all') return props.machines;
  const map: Record<string, string> = { idle: '空闲', working: '作业中', repair: '维修中' };
  return props.machines.filter((m) => m.status === map[filter.value]);
});
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-lg font-black">农机档案</h2>
      <el-select v-model="filter" placeholder="按状态筛选" size="small" class="w-36">
        <el-option label="全部" value="all" />
        <el-option label="空闲" value="idle" />
        <el-option label="作业中" value="working" />
        <el-option label="维修中" value="repair" />
      </el-select>
    </div>
    <el-table :data="filtered" size="small">
      <el-table-column prop="code" label="编号" width="130" />
      <el-table-column prop="name" label="农机" />
      <el-table-column prop="model" label="型号" width="110" />
      <el-table-column prop="horsepower" label="马力" width="80" />
      <el-table-column label="保养剩余工时" width="130">
        <template #default="{ row }">
          <span :class="row.maintenanceHours < 10 ? 'font-bold text-red-600' : 'text-slate-700'">
            {{ row.maintenanceHours }} h
          </span>
        </template>
      </el-table-column>
      <el-table-column label="绑定任务" min-width="150">
        <template #default="{ row }">
          <span v-if="row.taskId" class="text-emerald-700">{{ row.taskId }} · {{ row.currentTask }}</span>
          <span v-else class="text-slate-400">—</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="STATUS_COLORS[row.status]" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>
