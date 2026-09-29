<script setup lang="ts">
import { ref } from 'vue'
import type { EmailMetadata } from '../types'

const props = defineProps<{
  emails: EmailMetadata[]
  total: number
  loading: boolean
  selectedSender: string | null
  searchQuery: string
}>()

const emit = defineEmits<{
  (e: 'clear-sender'): void
  (e: 'update:search', query: string): void
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
    <div class="email-section-header">
      <div class="header-info">
        <h2>Email Messages</h2>
        <span class="count-pill">{{ total }} total</span>
      </div>

      <div class="email-search">
        <input
          type="text"
          placeholder="Filter subject or snippet..."
          :value="searchQuery"
          @input="emit('update:search', ($event.target as HTMLInputElement).value)"
          class="input-search"
        />
      </div>
    </div>

    <!-- Active Sender Filter Chip -->
    <div v-if="selectedSender" class="filter-chip-bar">
      <span>Filtered by sender: <strong>{{ selectedSender }}</strong></span>
      <button @click="emit('clear-sender')" class="btn-clear-filter">✕ Clear Filter</button>
    </div>

    <!-- Email List -->
    <div class="email-list">
      <div v-if="loading && emails.length === 0" class="loading-state">
        Loading emails...
      </div>
      <div v-else-if="emails.length === 0" class="empty-state">
        <p>No email messages found.</p>
        <p class="empty-hint" v-if="!selectedSender">
          Run an inbox scan to fetch and analyze your emails.
        </p>
      </div>
      <div
        v-else
        v-for="email in emails"
        :key="email.id"
        class="email-card"
        :class="{ expanded: expandedEmails[email.id] }"
        @click="toggleDetails(email.id)"
      >
        <div class="card-main">
          <div class="card-top">
            <div class="sender-info">
              <span class="direction-badge" :class="email.isSent ? 'sent' : 'received'">
                {{ email.isSent ? '⬆ Sent' : '⬇ Received' }}
              </span>
              <span class="sender-name">{{ email.fromName || email.from }}</span>
              <span class="sender-address" v-if="email.fromName">({{ email.from }})</span>
            </div>
            <div class="meta-right">
              <span class="email-size">{{ formatBytes(email.sizeEstimate) }}</span>
              <span class="email-date">{{ formatDate(email.date) }}</span>
            </div>
          </div>

          <div class="email-subject">
            {{ email.subject || '(No Subject)' }}
          </div>

          <div class="email-snippet">
            {{ email.snippet }}
          </div>
        </div>

        <!-- Expanded Details -->
        <div v-if="expandedEmails[email.id]" class="card-expanded-details" @click.stop>
          <div class="detail-row" v-if="email.to && email.to.length > 0">
            <strong>To:</strong> {{ email.to.join(', ') }}
          </div>
          <div class="detail-row">
            <strong>Message ID:</strong> <code>{{ email.id }}</code>
          </div>
          <div class="detail-row">
            <strong>Thread ID:</strong> <code>{{ email.threadId }}</code>
          </div>
          <div class="detail-row" v-if="email.labelIds && email.labelIds.length > 0">
            <strong>Labels:</strong>
            <span v-for="label in email.labelIds" :key="label" class="label-tag">
              {{ label }}
            </span>
          </div>
        </div>
      </div>

      <!-- Load More Button -->
      <div v-if="emails.length > 0 && emails.length < total" class="load-more-bar">
        <button @click="emit('load-more')" class="btn-load-more" :disabled="loading">
          {{ loading ? 'Loading...' : `Load More (${emails.length} of ${total})` }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.email-section {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.email-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1rem;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.header-info h2 {
  font-size: 1.25rem;
  font-weight: 700;
  color: #1a202c;
  margin: 0;
}

.count-pill {
  background: #edf2f7;
  color: #4a5568;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.2rem 0.55rem;
  border-radius: 12px;
}

.input-search {
  padding: 0.45rem 0.75rem;
  border: 1px solid #cbd5e0;
  border-radius: 6px;
  font-size: 0.85rem;
  min-width: 220px;
}

.filter-chip-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background: #ebf8ff;
  border: 1px solid #bee3f8;
  padding: 0.5rem 0.85rem;
  border-radius: 6px;
  margin-bottom: 1rem;
  font-size: 0.85rem;
  color: #2b6cb0;
}

.btn-clear-filter {
  background: transparent;
  border: none;
  color: #2b6cb0;
  font-weight: 700;
  cursor: pointer;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
}

.btn-clear-filter:hover {
  background: #bee3f8;
}

.email-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.email-card {
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 1rem;
  cursor: pointer;
  transition: all 0.15s ease;
  background: #ffffff;
}

.email-card:hover {
  border-color: #cbd5e0;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.04);
}

.email-card.expanded {
  border-color: #90cdf4;
  background: #f7fafc;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
}

.sender-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.direction-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}

.direction-badge.received {
  background: #ebf8ff;
  color: #2b6cb0;
}

.direction-badge.sent {
  background: #f0fff4;
  color: #276749;
}

.sender-name {
  font-weight: 600;
  color: #2d3748;
  font-size: 0.9rem;
}

.sender-address {
  color: #718096;
  font-size: 0.8rem;
}

.meta-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.8rem;
  color: #a0aec0;
}

.email-subject {
  font-weight: 600;
  color: #1a202c;
  font-size: 0.95rem;
  margin-bottom: 0.35rem;
}

.email-snippet {
  color: #4a5568;
  font-size: 0.85rem;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-expanded-details {
  margin-top: 0.85rem;
  padding-top: 0.85rem;
  border-top: 1px dashed #e2e8f0;
  font-size: 0.82rem;
  color: #4a5568;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.detail-row code {
  background: #edf2f7;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.78rem;
}

.label-tag {
  display: inline-block;
  background: #edf2f7;
  color: #4a5568;
  padding: 0.1rem 0.4rem;
  border-radius: 3px;
  margin-left: 0.3rem;
  font-size: 0.75rem;
}

.load-more-bar {
  text-align: center;
  margin-top: 1rem;
}

.btn-load-more {
  background: #edf2f7;
  border: 1px solid #cbd5e0;
  padding: 0.6rem 1.5rem;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  color: #2d3748;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-load-more:hover:not(:disabled) {
  background: #e2e8f0;
}

.btn-load-more:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 3rem;
  color: #718096;
}

.empty-hint {
  font-size: 0.85rem;
  color: #a0aec0;
  margin-top: 0.5rem;
}
</style>
