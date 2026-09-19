<script setup lang="ts">
import {
  ATTEMPT_ACTION_TEXT,
  ATTEMPT_RESULT_COLORS,
  ATTEMPT_RESULT_TEXT,
} from '../constants/app.constants';
import type { DispatchAttempt } from '../types/domain';

defineProps<{ attempts: DispatchAttempt[] }>();
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-1 text-lg font-black">派单记录（预占闭环留痕）</h2>
    <p class="mb-3 text-sm text-slate-500">
      每次派单/取消都会落库：成功占用、整单拒绝原因、释放与重复取消均可在此回读，刷新页面不丢失。
    </p>
    <el-empty v-if="attempts.length === 0" description="暂无派单记录" :image-size="60" />
    <ul v-else class="grid gap-2">
      <li
        v-for="attempt in attempts"
        :key="attempt.id"
        class="flex flex-wrap items-center gap-2 rounded-md border border-slate-100 px-3 py-2 text-sm"
      >
        <el-tag size="small" type="info" effect="plain">
          {{ ATTEMPT_ACTION_TEXT[attempt.action] ?? attempt.action }}
        </el-tag>
        <el-tag size="small" :type="ATTEMPT_RESULT_COLORS[attempt.result] ?? 'info'">
          {{ ATTEMPT_RESULT_TEXT[attempt.result] ?? attempt.result }}
        </el-tag>
        <span class="font-semibold">{{ attempt.taskId }} {{ attempt.taskType }} · {{ attempt.taskField }}</span>
        <span class="text-slate-500">{{ attempt.machineCode }} / {{ attempt.driverName }}</span>
        <span v-if="attempt.reason" class="font-medium text-red-600">原因：{{ attempt.reason }}</span>
        <span class="ml-auto text-xs text-slate-400">{{ attempt.createdAt }}</span>
      </li>
    </ul>
  </section>
</template>
