<script setup lang="ts">
import { ref } from 'vue'
import {
  AlertTriangle,
  ArrowDown,
  ArrowDownLeft,
  ArrowUpRight,
  ChevronDown,
  Filter,
  Inbox,
  Loader2,
  Mail,
  Search,
  Tag,
  X,
} from 'lucide-vue-next'
import type { EmailMetadata } from '../types'

const props = withDefaults(
  defineProps<{
    emails: EmailMetadata[]
    total: number
    loading: boolean
    selectedSender: string | null
    searchQuery: string
    filterMode?: 'all' | 'never_emailed'
  }>(),
  {
    filterMode: 'all',
  }
)

const emit = defineEmits<{
  (e: 'clear-sender'): void
  (e: 'update:search', query: string): void
  (e: 'update:filter-mode', mode: 'all' | 'never_emailed'): void
  (e: 'select-sender', sender: string): void
  (e: 'load-more'): void
}>()

const expandedEmails = ref<Record<string, boolean>>({})

const toggleDetails = (id: string) => {
  expandedEmails.value[id] = !expandedEmails.value[id]
}

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="email-section">
    <!-- Header with Title, Filter Tabs, and Search -->
    <div class="email-section-header">
      <div class="header-left">
        <div class="header-title-row">
          <Mail :size="20" class="header-icon" />
          <h2>Email Messages</h2>
          <span class="count-pill">{{ total }} total</span>
        </div>
        <p class="section-desc">
          Compact inbox preview highlighting incoming messages from people you never emailed back.
        </p>
      </div>

      <div class="header-controls">
        <!-- Quick Filter Mode Toggle -->
        <div class="email-filter-tabs">
          <button
            class="tab-chip"
            :class="{ active: filterMode === 'all' }"
            @click="emit('update:filter-mode', 'all')"
          >
            <Inbox :size="13" />
            <span>All Emails</span>
          </button>
          <button
            class="tab-chip alert-chip"
            :class="{ active: filterMode === 'never_emailed' }"
            @click="emit('update:filter-mode', 'never_emailed')"
          >
            <AlertTriangle :size="13" />
            <span>One-Way Senders Only</span>
          </button>
        </div>

        <!-- Search Input -->
        <div class="email-search">
          <div class="search-box">
            <Search :size="15" class="search-icon" />
            <input
              type="text"
              placeholder="Search subject, snippet, or sender..."
              :value="searchQuery"
              @input="emit('update:search', ($event.target as HTMLInputElement).value)"
              class="input-search"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Active Sender Filter Chip -->
    <div v-if="selectedSender" class="filter-chip-bar">
      <div class="filter-chip-text">
        <Filter :size="14" />
        <span
          >Filtered by sender: <strong>{{ selectedSender }}</strong></span
        >
      </div>
      <button @click="emit('clear-sender')" class="btn-clear-filter">
        <X :size="13" />
        <span>Clear Filter</span>
      </button>
    </div>

    <!-- One-Way Senders Filter Active Notice -->
    <div v-else-if="filterMode === 'never_emailed'" class="filter-chip-bar alert-notice">
      <div class="filter-chip-text">
        <AlertTriangle :size="14" />
        <span
          >Showing emails exclusively from senders you have
          <strong>never emailed back</strong>.</span
        >
      </div>
      <button @click="emit('update:filter-mode', 'all')" class="btn-clear-filter">
        <X :size="13" />
        <span>Show All</span>
      </button>
    </div>

    <!-- Email List -->
    <div class="email-list">
      <div v-if="loading && emails.length === 0" class="loading-state">
        <Loader2 :size="18" class="spinner-icon" />
        <span>Loading emails...</span>
      </div>
      <div v-else-if="emails.length === 0" class="empty-state">
        <p class="empty-title">No email messages found.</p>
        <p class="empty-hint" v-if="selectedSender">No messages match the current sender filter.</p>
        <p class="empty-hint" v-else-if="filterMode === 'never_emailed'">
          No emails from one-way senders found. Either you have replied to everyone or need to run
          an inbox scan.
        </p>
        <p class="empty-hint" v-else>Run an inbox scan above to fetch and analyze your emails.</p>
      </div>

      <!-- Compact Email Card List -->
      <div
        v-else
        v-for="email in emails"
        :key="email.id"
        class="email-card"
        :class="{
          expanded: expandedEmails[email.id],
          'never-emailed-card': email.senderNeverEmailed,
        }"
        @click="toggleDetails(email.id)"
      >
        <div class="card-main">
          <!-- Compact Row 1: Sender info, Never-Emailed badge, Size, Date -->
          <div class="card-top">
            <div class="sender-info">
              <span class="direction-badge" :class="email.isSent ? 'sent' : 'received'">
                <ArrowUpRight v-if="email.isSent" :size="12" />
                <ArrowDownLeft v-else :size="12" />
                <span>{{ email.isSent ? 'Sent' : 'Received' }}</span>
              </span>

              <!-- Highlight Badge for People You Never Emailed Back -->
              <span
                v-if="email.senderNeverEmailed"
                class="never-emailed-badge"
                title="You have never sent an email to this sender"
              >
                <AlertTriangle :size="12" />
                <span>Never Emailed Back</span>
                <span
                  v-if="email.senderEmailCount && email.senderEmailCount > 0"
                  class="badge-count"
                >
                  ({{ email.senderEmailCount }} received)
                </span>
              </span>

              <span class="sender-name">{{ email.fromName || email.from }}</span>
              <span class="sender-address" v-if="email.fromName">({{ email.from }})</span>
            </div>

            <div class="meta-right">
              <span class="email-size">{{ formatBytes(email.sizeEstimate) }}</span>
              <span class="email-date">{{ formatDate(email.date) }}</span>
              <ChevronDown :size="15" class="chevron" :class="{ open: expandedEmails[email.id] }" />
            </div>
          </div>

          <!-- Compact Row 2: Subject & Snippet Preview inline -->
          <div class="card-preview">
            <span class="email-subject">
              {{ email.subject || '(No Subject)' }}
            </span>
            <span class="snippet-separator" v-if="email.snippet">—</span>
            <span class="email-snippet" v-if="email.snippet">
              {{ email.snippet }}
            </span>
          </div>
        </div>

        <!-- Expanded Details Accordion -->
        <div v-if="expandedEmails[email.id]" class="card-expanded-details" @click.stop>
          <div class="expanded-grid">
            <div class="detail-row" v-if="email.to && email.to.length > 0">
              <span class="detail-label">To:</span>
              <span class="detail-value">{{ email.to.join(', ') }}</span>
            </div>

            <div class="detail-row" v-if="email.snippet">
              <span class="detail-label">Full Snippet:</span>
              <span class="detail-value snippet-full">{{ email.snippet }}</span>
            </div>

            <div class="detail-row">
              <span class="detail-label">Message ID:</span>
              <code class="detail-code">{{ email.id }}</code>
            </div>

            <div class="detail-row">
              <span class="detail-label">Thread ID:</span>
              <code class="detail-code">{{ email.threadId }}</code>
            </div>

            <div class="detail-row" v-if="email.labelIds && email.labelIds.length > 0">
              <span class="detail-label">Labels:</span>
              <div class="labels-container">
                <span v-for="label in email.labelIds" :key="label" class="label-tag">
                  <Tag :size="11" />
                  <span>{{ label }}</span>
                </span>
              </div>
            </div>

            <div class="detail-actions">
              <button
                v-if="!email.isSent && email.from"
                @click.stop="emit('select-sender', email.from)"
                class="btn-sender-filter"
              >
                <Search :size="13" />
                <span>Filter all emails from this sender ({{ email.from }})</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Load More Button -->
      <div v-if="emails.length > 0 && emails.length < total" class="load-more-bar">
        <button @click="emit('load-more')" class="btn-load-more" :disabled="loading">
          <ArrowDown :size="14" v-if="!loading" />
          <Loader2 :size="14" class="spinner-icon" v-else />
          <span>{{ loading ? 'Loading...' : `Load More (${emails.length} of ${total})` }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.email-section {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 1.25rem;
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.email-section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--color-border-subtle, #edf2f7);
}

.header-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.header-icon {
  color: var(--color-primary, #2563eb);
}

.header-title-row h2 {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
  margin: 0;
}

.count-pill {
  background: var(--color-surface-hover, #f1f5f9);
  color: var(--color-text-muted, #475569);
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--color-border, #e2e8f0);
  font-variant-numeric: tabular-nums;
}

.section-desc {
  font-size: 0.825rem;
  color: var(--color-text-muted, #64748b);
  margin-top: 0.2rem;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.email-filter-tabs {
  display: flex;
  background: #f1f5f9;
  padding: 3px;
  border-radius: 8px;
  gap: 2px;
}

.tab-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: none;
  background: transparent;
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-chip:hover {
  color: #0f172a;
}

.tab-chip.active {
  background: #ffffff;
  color: #0f172a;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

.tab-chip.alert-chip.active {
  background: #fffbeb;
  color: #b45309;
  box-shadow: 0 1px 2px rgba(217, 119, 6, 0.12);
}

.email-search {
  min-width: 240px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 0.65rem;
  color: var(--color-text-faint, #94a3b8);
  pointer-events: none;
}

.input-search {
  width: 100%;
  padding: 0.45rem 0.8rem 0.45rem 2rem;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  font-size: 0.85rem;
  background: var(--color-surface, #ffffff);
  color: var(--color-text-main, #0f172a);
}

.input-search:focus {
  outline: none;
  border-color: var(--color-primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
}

/* Filter Chip Bar */
.filter-chip-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.55rem 0.9rem;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: var(--radius-sm, 6px);
  font-size: 0.85rem;
  color: #1e40af;
}

.filter-chip-text {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.filter-chip-bar.alert-notice {
  background: #fffbeb;
  border-color: #fde68a;
  color: #92400e;
}

.btn-clear-filter {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  background: transparent;
  border: none;
  color: inherit;
  font-weight: 600;
  font-size: 0.8rem;
  cursor: pointer;
  text-decoration: underline;
}

/* Email List & Compact Items */
.email-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--color-text-muted, #64748b);
}

.empty-title {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--color-text-main, #0f172a);
}

.empty-hint {
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

/* Compact Email Card */
.email-card {
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  background: var(--color-surface, #ffffff);
  transition:
    background-color 0.15s ease,
    border-color 0.15s ease,
    box-shadow 0.15s ease;
  cursor: pointer;
  overflow: hidden;
}

.email-card:hover {
  background: #f8fafc;
  border-color: #cbd5e1;
}

/* Prominent Highlight for People You Never Emailed */
.email-card.never-emailed-card {
  border-left: 4px solid #f59e0b;
  background: #fffdfa;
}

.email-card.never-emailed-card:hover {
  background: #fffbeb;
}

.card-main {
  padding: 0.65rem 0.85rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

.sender-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
}

.direction-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  white-space: nowrap;
}

.direction-badge.received {
  background: #eff6ff;
  color: #1d4ed8;
}

.direction-badge.sent {
  background: #f0fdf4;
  color: #15803d;
}

/* Never-Emailed Pill */
.never-emailed-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: #fef3c7;
  color: #b45309;
  border: 1px solid #fde68a;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  white-space: nowrap;
}

.badge-count {
  font-weight: 600;
  opacity: 0.9;
}

.sender-name {
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--color-text-main, #0f172a);
}

.sender-address {
  font-size: 0.775rem;
  color: var(--color-text-muted, #64748b);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-right {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.775rem;
  color: var(--color-text-muted, #64748b);
  white-space: nowrap;
  flex-shrink: 0;
}

.email-size {
  font-variant-numeric: tabular-nums;
}

.email-date {
  font-variant-numeric: tabular-nums;
}

.chevron {
  transition: transform 0.2s ease;
  color: #94a3b8;
  flex-shrink: 0;
}

.chevron.open {
  transform: rotate(180deg);
}

/* Compact Preview Line (Subject + Snippet) */
.card-preview {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
  font-size: 0.85rem;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.email-subject {
  font-weight: 600;
  color: var(--color-text-main, #0f172a);
  flex-shrink: 0;
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.snippet-separator {
  color: #cbd5e1;
  flex-shrink: 0;
}

.email-snippet {
  color: var(--color-text-muted, #64748b);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.825rem;
}

/* Expanded Details */
.card-expanded-details {
  border-top: 1px dashed var(--color-border, #e2e8f0);
  background: #f8fafc;
  padding: 0.85rem 1rem;
}

.expanded-grid {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  font-size: 0.825rem;
}

.detail-row {
  display: flex;
  gap: 0.5rem;
  align-items: baseline;
}

.detail-label {
  font-weight: 600;
  color: #475569;
  min-width: 90px;
  flex-shrink: 0;
}

.detail-value {
  color: #1e293b;
  word-break: break-word;
}

.snippet-full {
  color: #334155;
  line-height: 1.4;
}

.detail-code {
  font-family: monospace;
  font-size: 0.775rem;
  background: #e2e8f0;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
}

.labels-container {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.label-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: #e0f2fe;
  color: #0369a1;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
}

.detail-actions {
  margin-top: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid #e2e8f0;
}

.btn-sender-filter {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  color: #1d4ed8;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.3rem 0.65rem;
  border-radius: 4px;
  cursor: pointer;
}

.btn-sender-filter:hover {
  background: #dbeafe;
}

/* Load More */
.load-more-bar {
  text-align: center;
  padding-top: 0.5rem;
}

.btn-load-more {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  color: var(--color-text-main, #0f172a);
  font-size: 0.85rem;
  font-weight: 600;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-sm, 6px);
  cursor: pointer;
  transition: all 0.15s ease;
}

.spinner-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.btn-load-more:hover:not(:disabled) {
  background: var(--color-surface-hover, #f1f5f9);
  border-color: var(--color-border-strong, #cbd5e1);
}
</style>
