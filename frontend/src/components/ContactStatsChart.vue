<script setup lang="ts">
import { computed } from 'vue'
import { Chart } from '@tanstack/vue-charts'
import { barX, defineChart } from '@tanstack/charts'
import { tooltip } from '@tanstack/charts/tooltip'
import { scaleBand, scaleLinear } from 'd3-scale'
import type { ContactStat } from '../types'

const props = withDefaults(
  defineProps<{
    contacts: ContactStat[]
    loading?: boolean
  }>(),
  {
    loading: false,
  }
)

interface ChartDatum {
  label: string
  count: number
}

const chartData = computed<ChartDatum[]>(() => {
  if (!props.contacts || props.contacts.length === 0) return []

  const topContacts = props.contacts.slice(0, 10)
  const rows: ChartDatum[] = []
  const usedLabels = new Set<string>()

  for (const c of topContacts) {
    const rawName = c.name?.trim() || ''
    const emailPrefix = c.email.split('@')[0]
    let label = rawName ? `${rawName} (${emailPrefix})` : c.email
    if (label.length > 28) {
      label =
        rawName && rawName !== emailPrefix
          ? `${rawName.slice(0, 14)}… (${emailPrefix})`
          : c.email.length > 28
            ? `${c.email.slice(0, 25)}…`
            : c.email
    }
    if (usedLabels.has(label)) {
      label = `${label} [${c.email}]`
    }
    usedLabels.add(label)

    rows.push({ label, count: c.receivedCount })
  }
  return rows
})

const chartDefinition = computed(() => {
  const data = chartData.value
  return defineChart({
    tooltip,
    focus: 'group-y',
    scales: {
      x: { scale: scaleLinear, nice: true, grid: true },
      y: { scale: () => scaleBand().padding(0.24) },
    },
    marks: [
      barX(data, {
        y: 'label',
        x: 'count',
        fill: '#3b82f6',
      }),
    ],
  })
})
</script>

<template>
  <div class="contacts-chart-card">
    <div class="chart-header">
      <h3 class="chart-title">Top Senders by Emails Received</h3>
    </div>

    <div v-if="loading" class="chart-status-message">Loading chart data...</div>
    <div v-else-if="chartData.length === 0" class="chart-status-message">
      No sender data available to plot.
    </div>
    <div v-else class="chart-container">
      <Chart
        :definition="chartDefinition"
        aria-label="Bar chart showing emails received from top senders you have never emailed"
        :style="{ height: `${Math.max(220, chartData.length * 36)}px` }"
      />
    </div>
  </div>
</template>

<style scoped>
.contacts-chart-card {
  background: var(--color-bg, #f8fafc);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 8px);
  padding: 1rem 1.25rem;
  margin-bottom: 1rem;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.chart-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
  margin: 0;
}

.chart-container {
  width: 100%;
  overflow-x: auto;
}

.chart-status-message {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-muted, #64748b);
  font-size: 0.875rem;
}
</style>
