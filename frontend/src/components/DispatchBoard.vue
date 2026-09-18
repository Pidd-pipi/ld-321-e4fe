<script setup lang="ts">
import { computed, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { ORDER_STATUS_ACTIVE, ORDER_STATUS_COLORS } from '../constants/app.constants';
import { cancelDispatchOrder } from '../services/dispatch.service';
import type { DispatchOrder } from '../types/domain';

const props = defineProps<{ orders: DispatchOrder[] }>();
const emit = defineEmits<{ changed: [] }>();

const cancellingId = ref('');

const activeOrders = computed(() => props.orders.filter((o) => o.status === ORDER_STATUS_ACTIVE));
const historyOrders = computed(() => props.orders.filter((o) => o.status !== ORDER_STATUS_ACTIVE));

const handleCancel = async (order: DispatchOrder) => {
  cancellingId.value = order.id;
  try {
    const result = await cancelDispatchOrder(order.id);
    ElMessage.success(result.message);
    emit('changed');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '取消派单失败');
  } finally {
    cancellingId.value = '';
  }
};

const formatTime = (value?: string) => (value ? value.replace('T', ' ').slice(0, 16) : '—');
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-lg font-black">派单预占看板</h2>
      <span class="text-xs text-slate-500">绑定关系与失败原因实时回读</span>
    </div>

    <h3 class="mb-2 text-sm font-bold text-slate-700">生效中的绑定（{{ activeOrders.length }}）</h3>
    <el-table :data="activeOrders" size="small" empty-text="暂无生效中的派单">
      <el-table-column label="任务" min-width="150">
        <template #default="{ row }">{{ row.taskType }} · {{ row.taskField }}</template>
      </el-table-column>
      <el-table-column prop="machineCode" label="农机" width="120" />
      <el-table-column prop="driverName" label="驾驶员" width="90" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="ORDER_STATUS_COLORS[row.status]" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="派单时间" width="140">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="110" fixed="right">
        <template #default="{ row }">
          <el-button
            size="small"
            type="danger"
            plain
            :loading="cancellingId === row.id"
            @click="handleCancel(row)"
          >取消派单</el-button>
        </template>
      </el-table-column>
    </el-table>

    <h3 class="mb-2 mt-4 text-sm font-bold text-slate-700">历史与失败记录（{{ historyOrders.length }}）</h3>
    <el-table :data="historyOrders" size="small" empty-text="暂无历史派单记录">
      <el-table-column label="任务" min-width="150">
        <template #default="{ row }">{{ row.taskType }} · {{ row.taskField }}</template>
      </el-table-column>
      <el-table-column label="农机 / 驾驶员" min-width="170">
        <template #default="{ row }">{{ row.machineCode || '—' }} / {{ row.driverName || '—' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="ORDER_STATUS_COLORS[row.status]" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="失败原因 / 说明" min-width="220">
        <template #default="{ row }">
          <span v-if="row.failReason" class="text-red-600">{{ row.failReason }}</span>
          <span v-else class="text-slate-500">已于 {{ formatTime(row.cancelledAt) }} 取消并释放资源</span>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>
