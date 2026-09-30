<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Clock,
  HardDrive,
  Mail,
  Pause,
  Play,
  RotateCw,
  Users,
  UserX,
} from 'lucide-vue-next'
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

const progressPercent = computed(() => {
  if (!props.progress.estimatedTotal || props.progress.estimatedTotal <= 0) {
    return 0
  }
  const pct = Math.round((props.progress.totalEmails / props.progress.estimatedTotal) * 100)
  return Math.min(100, Math.max(0, pct))
})

const neverEmailedPercent = computed(() => {
  if (!props.stats.uniqueSenders || props.stats.uniqueSenders === 0) return 0
  return Math.round((props.stats.neverEmailedCount / props.stats.uniqueSenders) * 100)
})
</script>

<template>
  <div class="dashboard-panel">
    <!-- Stat Cards -->
    <div class="stats-grid">
      <!-- High-Priority: Highlighted One-Way Senders Card -->
      <div
        class="stat-card highlight-card"
        @click="emit('select-tab', 'never_emailed')"
        title="Click to view senders you have never emailed back"
      >
        <div class="stat-header-row">
          <div class="stat-label-group">
            <UserX :size="16" class="stat-card-icon warning-icon" />
            <span class="stat-label">Never Emailed Back</span>
          </div>
          <span class="alert-pill">Key Focus</span>
        </div>
        <div class="stat-value warning-text">{{ stats.neverEmailedCount.toLocaleString() }}</div>
        <div class="stat-sub warning-sub">
          <span>{{ neverEmailedPercent }}% of senders</span>
          <span class="view-link">
            <span>View all</span>
            <ArrowRight :size="13" />
          </span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-header-row">
          <div class="stat-label-group">
            <Mail :size="16" class="stat-card-icon" />
            <span class="stat-label">Total Processed</span>
          </div>
        </div>
        <div class="stat-value">{{ stats.totalEmailsProcessed.toLocaleString() }}</div>
        <div class="stat-sub">
          <span class="stat-sub-item">
            <ArrowDown :size="12" class="arrow-received" />
            {{ stats.totalReceived.toLocaleString() }} received
          </span>
          <span class="stat-sub-item">
            <ArrowUp :size="12" class="arrow-sent" />
            {{ stats.totalSent.toLocaleString() }} sent
          </span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-header-row">
          <div class="stat-label-group">
            <Users :size="16" class="stat-card-icon" />
            <span class="stat-label">Unique Senders</span>
          </div>
        </div>
        <div class="stat-value">{{ stats.uniqueSenders.toLocaleString() }}</div>
        <div class="stat-sub">Distinct contacts analyzed</div>
      </div>

      <div class="stat-card">
        <div class="stat-header-row">
          <div class="stat-label-group">
            <HardDrive :size="16" class="stat-card-icon" />
            <span class="stat-label">Total Size</span>
          </div>
        </div>
        <div class="stat-value">{{ formatBytes(stats.totalSizeBytes) }}</div>
        <div class="stat-sub">Estimated space analyzed</div>
      </div>
    </div>

    <!-- Scan Controls & Progress Bar Card -->
    <div class="scan-panel" :class="{ 'is-active': isScanning }">
      <div class="scan-bar-header">
        <div class="scan-info">
          <div class="status-indicator">
            <span class="status-dot" :class="{ pulsing: isScanning }"></span>
            <span class="status-text">{{
              progress.statusMessage || (isScanning ? 'Scanning...' : 'Ready')
            }}</span>
          </div>
          <div class="last-sync" v-if="progress.lastScannedAt">
            <Clock :size="12" />
            <span>Last scanned: {{ formattedLastScanned }}</span>
          </div>
        </div>

        <div class="scan-actions">
          <button v-if="!isScanning" @click="emit('start-scan', false)" class="btn btn-primary">
            <Play :size="15" />
            {{ stats.totalEmailsProcessed > 0 ? 'Resume Scan' : 'Start Inbox Scan' }}
          </button>

          <button v-if="isScanning" @click="emit('pause-scan')" class="btn btn-warning">
            <Pause :size="15" />
            Pause Scan
          </button>

          <button
            v-if="!isScanning && stats.totalEmailsProcessed > 0"
            @click="emit('start-scan', true)"
            class="btn btn-outline"
            title="Restart scan from the newest messages"
          >
            <RotateCw :size="14" />
            Rescan All
          </button>
        </div>
      </div>

      <!-- Scan Progress Bar Section (Visible during scan or after progress) -->
      <div class="progress-section" v-if="isScanning || progress.totalEmails > 0">
        <div class="progress-meta">
          <span class="progress-label">
            <template v-if="isScanning"> Scanning in progress... </template>
            <template v-else> Scan status </template>
          </span>
          <div class="progress-numbers">
            <template v-if="progress.estimatedTotal && progress.estimatedTotal > 0">
              <span class="numbers-count">
                <strong>{{ progress.totalEmails.toLocaleString() }}</strong> /
                {{ progress.estimatedTotal.toLocaleString() }} emails
              </span>
              <span class="numbers-percent">{{ progressPercent }}%</span>
            </template>
            <template v-else>
              <span class="numbers-count">
                <strong>{{ progress.totalEmails.toLocaleString() }}</strong> emails scanned
              </span>
            </template>
          </div>
        </div>

        <div class="progress-track">
          <!-- Determinate bar when estimatedTotal is known -->
          <div
            v-if="progress.estimatedTotal && progress.estimatedTotal > 0"
            class="progress-fill determinate"
            :class="{ active: isScanning }"
            :style="{ width: `${Math.max(isScanning ? 4 : 0, progressPercent)}%` }"
          ></div>
          <!-- Indeterminate animated shimmer bar when estimatedTotal is 0 or calculating -->
          <div
            v-else
            class="progress-fill indeterminate"
            :class="{ active: isScanning }"
            :style="{ width: isScanning ? '100%' : '100%' }"
          ></div>
        </div>
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
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 1.25rem 1.25rem 1rem;
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  transition:
    transform 0.15s ease,
    box-shadow 0.15s ease;
}

.stat-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stat-label-group {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.stat-card-icon {
  color: var(--color-text-faint, #94a3b8);
  flex-shrink: 0;
}

.stat-card-icon.warning-icon {
  color: #d97706;
}

.stat-label {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted, #64748b);
}

.stat-value {
  font-size: 2rem;
  font-weight: 800;
  color: var(--color-text-main, #0f172a);
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
  margin: 0.2rem 0;
}

.stat-sub {
  font-size: 0.825rem;
  color: var(--color-text-muted, #64748b);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

/* Highlighting the One-Way Senders Card */
.highlight-card {
  background: #fffbeb;
  border: 2px solid #fde68a;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.highlight-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 14px rgba(217, 119, 6, 0.15);
  border-color: #f59e0b;
}

.alert-pill {
  background: #fef3c7;
  color: #b45309;
  border: 1px solid #fde68a;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
}

.warning-text {
  color: #d97706;
}

.warning-sub {
  color: #92400e;
  font-weight: 600;
}

.view-link {
  color: #b45309;
  text-decoration: underline;
  text-underline-offset: 2px;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.stat-sub-item {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.arrow-received {
  color: var(--color-primary, #2563eb);
}

.arrow-sent {
  color: var(--color-warning-accent, #ea580c);
}

/* Scan Panel & Progress Bar */
.scan-panel {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 1.25rem;
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 1rem;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.scan-panel.is-active {
  border-color: #93c5fd;
  box-shadow:
    0 0 0 1px #bfdbfe,
    var(--shadow-sm);
}

.scan-bar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
  gap: 0.6rem;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #94a3b8;
}

.status-dot.pulsing {
  background: #2563eb;
  box-shadow: 0 0 0 0 rgba(37, 99, 235, 0.7);
  animation: pulse-ring 1.5s cubic-bezier(0.24, 0, 0.38, 1) infinite;
}

@keyframes pulse-ring {
  0% {
    box-shadow: 0 0 0 0 rgba(37, 99, 235, 0.7);
  }
  70% {
    box-shadow: 0 0 0 7px rgba(37, 99, 235, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(37, 99, 235, 0);
  }
}

.status-text {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--color-text-main, #0f172a);
}

.last-sync {
  font-size: 0.8rem;
  color: var(--color-text-muted, #64748b);
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.scan-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

/* Progress bar styling */
.progress-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: var(--radius-sm, 6px);
  padding: 0.85rem 1rem;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
}

.progress-label {
  font-weight: 600;
  color: #334155;
}

.progress-numbers {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.numbers-count {
  color: #475569;
  font-variant-numeric: tabular-nums;
}

.numbers-percent {
  font-weight: 700;
  color: #2563eb;
  background: #dbeafe;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
}

.progress-track {
  width: 100%;
  height: 10px;
  background: #e2e8f0;
  border-radius: 999px;
  overflow: hidden;
  position: relative;
}

.progress-fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.3s ease;
}

.progress-fill.determinate {
  background: linear-gradient(90deg, #3b82f6 0%, #2563eb 100%);
}

.progress-fill.determinate.active {
  background: linear-gradient(
    45deg,
    #2563eb 25%,
    #3b82f6 25%,
    #3b82f6 50%,
    #2563eb 50%,
    #2563eb 75%,
    #3b82f6 75%,
    #3b82f6 100%
  );
  background-size: 24px 24px;
  animation: move-stripes 1.2s linear infinite;
}

.progress-fill.indeterminate.active {
  background: linear-gradient(90deg, #bfdbfe 0%, #2563eb 50%, #bfdbfe 100%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes move-stripes {
  0% {
    background-position: 0 0;
  }
  100% {
    background-position: 24px 0;
  }
}

@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
</style>
