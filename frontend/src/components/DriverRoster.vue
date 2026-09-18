<script setup lang="ts">
import { STATUS_COLORS } from '../constants/app.constants';
import type { Driver } from '../types/domain';

defineProps<{ drivers: Driver[] }>();
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">驾驶员管理</h2>
    <div class="grid gap-3 md:grid-cols-3">
      <article
        v-for="driver in drivers"
        :key="driver.id"
        class="rounded-md border"
        :class="driver.status !== '在岗' ? 'border-slate-200 bg-slate-50' : 'border-slate-200'"
      >
        <div class="flex items-center justify-between">
          <strong>{{ driver.name }}</strong>
          <el-rate :model-value="driver.rating" disabled size="small" />
        </div>
        <p class="mt-2 text-sm text-slate-600">{{ driver.licenseNo }}</p>
        <p class="text-sm text-slate-600">{{ driver.phone }}</p>
        <p class="mt-2 text-sm text-slate-600">
          {{ driver.shift }} · 休 {{ driver.restDay }} · {{ driver.monthAreaMu }} 亩
        </p>
        <div class="mt-3 flex flex-wrap items-center gap-2">
          <el-tag :type="STATUS_COLORS[driver.status]" size="small">当日{{ driver.status }}</el-tag>
          <el-tag :type="STATUS_COLORS[driver.workStatus]" size="small" effect="plain">
            {{ driver.workStatus === '作业中' ? `作业中 · ${driver.taskId}` : '空闲' }}
          </el-tag>
        </div>
      </article>
    </div>
  </section>
</template>
