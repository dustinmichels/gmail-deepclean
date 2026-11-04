<script setup lang="ts">
import { ref } from 'vue'
import { ArrowUpDown, Check, Copy, Download, Mail, Search, UserX } from 'lucide-vue-next'
import type { ContactStat } from '../types'
import ContactStatsChart from './ContactStatsChart.vue'

withDefaults(
  defineProps<{
    contacts: ContactStat[]
    total: number
    loading: boolean
    error?: string | null
    searchQuery: string
    sortField: string
  }>(),
  {
    error: null,
  }
)

const emit = defineEmits<{
  (e: 'update:search', search: string): void
  (e: 'update:sort', sort: string): void
  (e: 'select-sender', email: string): void
  (e: 'export-csv'): void
  (e: 'retry'): void
}>()
const copiedEmail = ref<string | null>(null)

const copyToClipboard = async (email: string) => {
  try {
    await navigator.clipboard.writeText(email)
    copiedEmail.value = email
    setTimeout(() => {
      if (copiedEmail.value === email) {
        copiedEmail.value = null
      }
    }, 2000)
  } catch (err) {
    console.error('Failed to copy: ', err)
  }
}

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formatDate = (dateStr?: string) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="sender-section">
    <div class="section-header">
      <div class="header-left">
        <UserX :size="20" class="header-icon" />
        <h2>Senders You've Never Emailed</h2>
        <span class="total-badge">{{ total }} senders</span>
      </div>

      <div class="header-actions">
        <button
          @click="emit('export-csv')"
          class="btn-export"
          :disabled="contacts.length === 0"
          title="Export current view to CSV"
        >
          <Download :size="14" />
          <span>Export CSV</span>
        </button>
      </div>
    </div>

    <p class="section-desc">
      Anyone you have sent an email to is treated as a real contact and hidden here.
    </p>

    <!-- Search & Sort Controls -->
    <div class="controls-bar">
      <div class="search-box">
        <Search :size="15" class="search-icon" />
        <input
          type="text"
          placeholder="Search email or name..."
          :value="searchQuery"
          @input="emit('update:search', ($event.target as HTMLInputElement).value)"
          class="input-search"
        />
      </div>

      <div class="sort-box">
        <ArrowUpDown :size="14" class="sort-icon" />
        <label>Sort:</label>
        <select
          :value="sortField"
          @change="emit('update:sort', ($event.target as HTMLSelectElement).value)"
          class="select-sort"
        >
          <option value="received_count">Most Received</option>
          <option value="total_size">Largest Mailbox Space</option>
          <option value="last_received">Most Recent</option>
          <option value="email">Email Address</option>
        </select>
      </div>
    </div>

    <ContactStatsChart :contacts="contacts" :loading="loading" />

    <!-- Contacts Table -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">Loading senders...</div>
      <div v-else-if="error" class="error-state">
        <p class="error-message">{{ error }}</p>
        <button @click="emit('retry')" class="btn-retry">Retry</button>
      </div>
      <div v-else-if="contacts.length === 0" class="empty-state">
        <p>No senders found.</p>
      </div>
      <table v-else class="sender-table">
        <thead>
          <tr>
            <th>Sender</th>
            <th>Received</th>
            <th title="Total space occupied in your Gmail mailbox by messages from this sender">
              Mailbox Size
            </th>
            <th>Last Received</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in contacts" :key="c.email">
            <td class="col-contact">
              <div class="contact-name" v-if="c.name">{{ c.name }}</div>
              <div class="contact-email">{{ c.email }}</div>
            </td>

            <td class="col-count">
              <span class="badge received-badge">{{ c.receivedCount }}</span>
            </td>

            <td class="col-size">{{ formatBytes(c.totalSize) }}</td>

            <td class="col-date">{{ formatDate(c.lastReceived) }}</td>

            <td class="col-actions">
              <button
                @click="emit('select-sender', c.email)"
                class="action-btn view-btn"
                title="Filter emails from this sender"
              >
                <Mail :size="13" />
                <span>Emails</span>
              </button>
              <button
                @click="copyToClipboard(c.email)"
                class="action-btn copy-btn"
                :class="{ copied: copiedEmail === c.email }"
                :title="copiedEmail === c.email ? 'Copied!' : 'Copy email address'"
              >
                <Check v-if="copiedEmail === c.email" :size="13" />
                <Copy v-else :size="13" />
                <span>{{ copiedEmail === c.email ? 'Copied' : 'Copy' }}</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.error-state {
  text-align: center;
  padding: 3rem 1rem;
  color: #dc2626;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.error-message {
  font-size: 0.9rem;
  margin: 0;
}

.btn-retry {
  background: #ffffff;
  border: 1px solid #dc2626;
  color: #dc2626;
  border-radius: var(--radius-sm, 6px);
  padding: 0.35rem 0.85rem;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
}

.btn-retry:hover {
  background: #fef2f2;
}
.sender-section {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 1.25rem;
  box-shadow: var(--shadow-sm);
  margin-bottom: 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.header-icon {
  color: var(--color-primary, #2563eb);
}

.header-left h2 {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
  margin: 0;
}
.total-badge {
  background: var(--color-surface-hover, #f1f5f9);
  color: var(--color-text-muted, #475569);
  font-size: 0.8rem;
  font-weight: 700;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  border: 1px solid var(--color-border, #e2e8f0);
}

.btn-export {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #cbd5e1);
  padding: 0.45rem 0.9rem;
  border-radius: var(--radius-sm, 6px);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  color: var(--color-text-main, #0f172a);
  transition: all 0.15s ease;
}

.btn-export:hover:not(:disabled) {
  background: var(--color-surface-hover, #f1f5f9);
  border-color: var(--color-border-strong, #94a3b8);
}

.btn-export:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.section-desc {
  margin: -0.5rem 0 0;
  font-size: 0.85rem;
  color: var(--color-text-muted, #64748b);
}

.controls-bar {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
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
  padding: 0.45rem 0.75rem 0.45rem 2rem;
  border: 1px solid var(--color-border, #cbd5e1);
  border-radius: var(--radius-sm, 6px);
  font-size: 0.85rem;
  min-width: 220px;
}

.input-search:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
}

.sort-box {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.85rem;
  color: #475569;
}

.sort-icon {
  color: var(--color-text-muted, #64748b);
}

.select-sort {
  padding: 0.45rem 0.75rem;
  border: 1px solid var(--color-border, #cbd5e1);
  border-radius: var(--radius-sm, 6px);
  font-size: 0.85rem;
  background: #ffffff;
  color: #0f172a;
}

/* Table styling */
.table-container {
  overflow-x: auto;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
}

.sender-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 0.875rem;
}

.sender-table th {
  background: #f8fafc;
  color: #475569;
  font-weight: 700;
  font-size: 0.775rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e2e8f0;
}

.sender-table td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: middle;
}

.sender-table tbody tr:hover {
  background: #f8fafc;
}

.col-contact {
  max-width: 260px;
}

.contact-name {
  font-weight: 600;
  color: var(--color-text-main, #0f172a);
}

.contact-email {
  color: #64748b;
  font-size: 0.8rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badge {
  display: inline-block;
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.received-badge {
  background: #eff6ff;
  color: #1d4ed8;
}

.col-size {
  color: #475569;
  font-size: 0.825rem;
  font-variant-numeric: tabular-nums;
}

.col-date {
  color: #64748b;
  font-size: 0.825rem;
  white-space: nowrap;
}

.col-actions {
  white-space: nowrap;
  display: flex;
  gap: 0.4rem;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.35rem 0.65rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
  border: 1px solid #cbd5e1;
  background: #ffffff;
  cursor: pointer;
  transition: all 0.15s ease;
}

.copy-btn.copied {
  color: #166534;
  border-color: #bbf7d0;
  background: #f0fdf4;
}

.action-btn:hover {
  background: #f1f5f9;
}

.view-btn {
  color: #2563eb;
  border-color: #bfdbfe;
  background: #eff6ff;
}

.view-btn:hover {
  background: #dbeafe;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2.5rem;
  color: #64748b;
}
</style>
