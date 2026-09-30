<script setup lang="ts">
import { computed } from 'vue'
import { Chart } from '@tanstack/vue-charts'
import { barX, defineChart, group } from '@tanstack/charts'
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
  direction: 'They Sent Me' | 'I Sent Them'
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

    rows.push(
      { label, direction: 'They Sent Me', count: c.receivedCount },
      { label, direction: 'I Sent Them', count: c.sentCount }
    )
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
        z: 'direction',
        fill: (d: ChartDatum) => (d.direction === 'They Sent Me' ? '#3b82f6' : '#10b981'),
        layout: group({ padding: 0.1 }),
      }),
    ],
  })
})
</script>

<template>
  <div class="contacts-chart-card">
    <div class="chart-header">
      <h3 class="chart-title">Emails Received vs. Sent by Contact</h3>
      <div class="chart-legend" v-if="chartData.length > 0">
        <div class="legend-item">
          <span class="swatch swatch-received"></span>
          <span>They Sent Me (Received)</span>
        </div>
        <div class="legend-item">
          <span class="swatch swatch-sent"></span>
          <span>I Sent Them</span>
        </div>
      </div>
    </div>

    <div v-if="loading" class="chart-status-message">
      Loading chart data...
    </div>
    <div v-else-if="chartData.length === 0" class="chart-status-message">
      No contact data available to plot.
    </div>
    <div v-else class="chart-container">
      <Chart
        :definition="chartDefinition"
        aria-label="Grouped bar chart showing emails received versus emails sent for top contacts"
        :style="{ height: `${Math.max(220, (chartData.length / 2) * 36)}px` }"
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

.chart-legend {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 0.78rem;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--color-text-muted, #64748b);
  font-weight: 500;
}

.swatch {
  width: 12px;
  height: 12px;
  border-radius: 2px;
}

.swatch-received {
  background: #3b82f6;
}

.swatch-sent {
  background: #10b981;
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
