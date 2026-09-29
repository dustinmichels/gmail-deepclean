<script setup lang="ts">
import { computed } from 'vue'
import type { OverallStats, ScanProgress } from '../types'

const props = defineProps<{
  progress: ScanProgress
  stats: OverallStats
  isScanning: boolean
}>()

const emit = defineEmits<{
  (e: 'start-scan', reset: boolean): void
  (e: 'pause-scan'): void
  (e: 'select-tab', tab: string): void
}>()

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formattedLastScanned = computed(() => {
  if (!props.progress.lastScannedAt) return 'Never'
  const date = new Date(props.progress.lastScannedAt)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
})
</script>

<template>
  <div class="dashboard-panel">
    <!-- Stat Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Total Processed</div>
        <div class="stat-value">{{ stats.totalEmailsProcessed.toLocaleString() }}</div>
        <div class="stat-sub">
          <span>⬇ {{ stats.totalReceived }} received</span>
          <span>⬆ {{ stats.totalSent }} sent</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-label">Unique Senders</div>
        <div class="stat-value">{{ stats.uniqueSenders.toLocaleString() }}</div>
        <div class="stat-sub">Addresses found</div>
      </div>

      <div
        class="stat-card highlight-card"
        @click="emit('select-tab', 'never_emailed')"
        title="Click to view senders you have never emailed"
      >
        <div class="stat-label">Never Emailed Back</div>
        <div class="stat-value warning-text">{{ stats.neverEmailedCount.toLocaleString() }}</div>
        <div class="stat-sub warning-sub">Senders you never emailed ➔</div>
      </div>

      <div class="stat-card">
        <div class="stat-label">Total Size</div>
        <div class="stat-value">{{ formatBytes(stats.totalSizeBytes) }}</div>
        <div class="stat-sub">Estimated space</div>
      </div>
    </div>

    <!-- Scan Controls & Progress Bar -->
    <div class="scan-bar">
      <div class="scan-info">
        <div class="status-indicator">
          <span class="status-dot" :class="{ pulsing: isScanning }"></span>
          <span class="status-text">{{ progress.statusMessage || 'Ready' }}</span>
        </div>
        <div class="last-sync" v-if="progress.lastScannedAt">
          Last scanned: {{ formattedLastScanned }}
        </div>
      </div>

      <div class="scan-actions">
        <button
          v-if="!isScanning"
          @click="emit('start-scan', false)"
          class="btn btn-primary"
        >
          {{ stats.totalEmailsProcessed > 0 ? 'Resume Scan' : 'Start Inbox Scan' }}
        </button>

        <button
          v-if="isScanning"
          @click="emit('pause-scan')"
          class="btn btn-warning"
        >
          Pause Scan
        </button>

        <button
          v-if="!isScanning && stats.totalEmailsProcessed > 0"
          @click="emit('start-scan', true)"
          class="btn btn-outline"
          title="Restart scan from the newest messages"
        >
          Rescan All
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.highlight-card {
  cursor: pointer;
  border-color: #fbd38d;
  background: #fffaf0;
}

.highlight-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 6px rgba(221, 107, 32, 0.15);
}

.stat-label {
  font-size: 0.85rem;
  color: #718096;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.35rem;
}

.stat-value {
  font-size: 1.85rem;
  font-weight: 700;
  color: #1a202c;
  line-height: 1.2;
}

.warning-text {
  color: #dd6b20;
}

.stat-sub {
  margin-top: 0.5rem;
  font-size: 0.8rem;
  color: #a0aec0;
  display: flex;
  gap: 0.75rem;
}

.warning-sub {
  color: #c05621;
  font-weight: 600;
}

.scan-bar {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1rem 1.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
}

.scan-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #2d3748;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #a0aec0;
}

.status-dot.pulsing {
  background: #38a169;
  box-shadow: 0 0 0 0 rgba(56, 161, 105, 0.7);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(56, 161, 105, 0.7);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 8px rgba(56, 161, 105, 0);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(56, 161, 105, 0);
  }
}

.last-sync {
  font-size: 0.8rem;
  color: #a0aec0;
}

.scan-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.btn {
  border: none;
  padding: 0.55rem 1.15rem;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-primary {
  background: #3182ce;
  color: white;
}

.btn-primary:hover {
  background: #2b6cb0;
}

.btn-warning {
  background: #dd6b20;
  color: white;
}

.btn-warning:hover {
  background: #c05621;
}

.btn-outline {
  background: transparent;
  border: 1px solid #cbd5e0;
  color: #4a5568;
}

.btn-outline:hover {
  background: #edf2f7;
}
</style>
