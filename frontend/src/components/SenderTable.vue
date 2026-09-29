<script setup lang="ts">
import { ref } from 'vue'
import type { ContactStat } from '../types'

const props = defineProps<{
  contacts: ContactStat[]
  total: number
  loading: boolean
  activeFilter: string
  searchQuery: string
  sortField: string
  sortOrder: string
  neverEmailedCount: number
}>()

const emit = defineEmits<{
  (e: 'update:filter', filter: string): void
  (e: 'update:search', search: string): void
  (e: 'update:sort', sort: string): void
  (e: 'select-sender', email: string): void
  (e: 'export-csv'): void
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
        <h2>Sender Analysis</h2>
        <span class="total-badge">{{ total }} contacts</span>
      </div>

      <div class="header-actions">
        <button
          @click="emit('export-csv')"
          class="btn-export"
          :disabled="contacts.length === 0"
          title="Export current view to CSV"
        >
          📥 Export CSV
        </button>
      </div>
    </div>

    <!-- Filter Tabs & Search Controls -->
    <div class="controls-bar">
      <div class="tabs">
        <button
          class="tab-btn"
          :class="{ active: activeFilter === 'all' }"
          @click="emit('update:filter', 'all')"
        >
          All Senders
        </button>

        <button
          class="tab-btn never-emailed-tab"
          :class="{ active: activeFilter === 'never_emailed' }"
          @click="emit('update:filter', 'never_emailed')"
        >
          ⚠ Never Emailed Back
          <span class="pill" v-if="neverEmailedCount > 0">{{ neverEmailedCount }}</span>
        </button>

        <button
          class="tab-btn"
          :class="{ active: activeFilter === 'emailed' }"
          @click="emit('update:filter', 'emailed')"
        >
          Reciprocal Contacts
        </button>
      </div>

      <div class="filter-inputs">
        <div class="search-box">
          <input
            type="text"
            placeholder="Search email or name..."
            :value="searchQuery"
            @input="emit('update:search', ($event.target as HTMLInputElement).value)"
            class="input-search"
          />
        </div>

        <div class="sort-box">
          <label>Sort:</label>
          <select
            :value="sortField"
            @change="emit('update:sort', ($event.target as HTMLSelectElement).value)"
            class="select-sort"
          >
            <option value="received_count">Most Received</option>
            <option value="sent_count">Most Sent</option>
            <option value="total_size">Largest Size</option>
            <option value="last_received">Most Recent</option>
            <option value="email">Email Address</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Contacts Table -->
    <div class="table-container">
      <div v-if="loading" class="loading-state">Loading contacts...</div>
      <div v-else-if="contacts.length === 0" class="empty-state">
        <p>No contacts found matching the selected filter.</p>
      </div>
      <table v-else class="sender-table">
        <thead>
          <tr>
            <th>Contact / Sender</th>
            <th>Received</th>
            <th>Sent Back</th>
            <th>Total Size</th>
            <th>Last Received</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="c in contacts"
            :key="c.email"
            :class="{ 'row-warning': c.neverEmailed }"
          >
            <td class="col-contact">
              <div class="contact-name" v-if="c.name">{{ c.name }}</div>
              <div class="contact-email">{{ c.email }}</div>
            </td>

            <td class="col-count">
              <span class="badge received-badge">{{ c.receivedCount }}</span>
            </td>

            <td class="col-count">
              <span v-if="c.sentCount > 0" class="badge sent-badge">{{ c.sentCount }}</span>
              <span v-else class="badge warning-badge" title="You have never sent an email to this address">
                Never
              </span>
            </td>

            <td class="col-size">{{ formatBytes(c.totalSize) }}</td>

            <td class="col-date">{{ formatDate(c.lastReceived) }}</td>

            <td class="col-actions">
              <button
                @click="emit('select-sender', c.email)"
                class="action-btn view-btn"
                title="Filter emails from this sender"
              >
                Emails
              </button>
              <button
                @click="copyToClipboard(c.email)"
                class="action-btn copy-btn"
                :title="copiedEmail === c.email ? 'Copied!' : 'Copy email address'"
              >
                {{ copiedEmail === c.email ? '✓' : 'Copy' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.sender-section {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  margin-bottom: 2rem;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.header-left h2 {
  font-size: 1.25rem;
  font-weight: 700;
  color: #1a202c;
  margin: 0;
}

.total-badge {
  background: #edf2f7;
  color: #4a5568;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.2rem 0.55rem;
  border-radius: 12px;
}

.btn-export {
  background: #edf2f7;
  border: 1px solid #cbd5e0;
  padding: 0.4rem 0.85rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  color: #2d3748;
  transition: all 0.15s ease;
}

.btn-export:hover:not(:disabled) {
  background: #e2e8f0;
}

.btn-export:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.controls-bar {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

@media (min-width: 768px) {
  .controls-bar {
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
  }
}

.tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tab-btn {
  background: #edf2f7;
  border: 1px solid transparent;
  color: #4a5568;
  padding: 0.5rem 0.9rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  background: #e2e8f0;
}

.tab-btn.active {
  background: #3182ce;
  color: white;
}

.never-emailed-tab.active {
  background: #dd6b20;
}

.never-emailed-tab .pill {
  background: #c05621;
  color: white;
  font-size: 0.75rem;
  padding: 0.1rem 0.4rem;
  border-radius: 10px;
}

.filter-inputs {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
}

.input-search {
  padding: 0.45rem 0.75rem;
  border: 1px solid #cbd5e0;
  border-radius: 6px;
  font-size: 0.85rem;
  min-width: 200px;
}

.sort-box {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  color: #4a5568;
}

.select-sort {
  padding: 0.45rem 0.65rem;
  border: 1px solid #cbd5e0;
  border-radius: 6px;
  font-size: 0.85rem;
  background: white;
}

.table-container {
  overflow-x: auto;
}

.sender-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 0.9rem;
}

.sender-table th {
  background: #f7fafc;
  color: #718096;
  font-weight: 600;
  padding: 0.75rem 1rem;
  border-bottom: 2px solid #e2e8f0;
}

.sender-table td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #edf2f7;
  vertical-align: middle;
}

.row-warning {
  background-color: #fffaf0;
}

.col-contact {
  max-width: 260px;
}

.contact-name {
  font-weight: 600;
  color: #2d3748;
}

.contact-email {
  color: #718096;
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
  font-weight: 600;
}

.received-badge {
  background: #ebf8ff;
  color: #2b6cb0;
}

.sent-badge {
  background: #f0fff4;
  color: #276749;
}

.warning-badge {
  background: #feebc8;
  color: #c05621;
}

.col-size {
  color: #4a5568;
  font-size: 0.85rem;
}

.col-date {
  color: #718096;
  font-size: 0.85rem;
  white-space: nowrap;
}

.col-actions {
  white-space: nowrap;
  display: flex;
  gap: 0.4rem;
}

.action-btn {
  padding: 0.35rem 0.65rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
  border: 1px solid #cbd5e0;
  background: white;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: #edf2f7;
}

.view-btn {
  color: #3182ce;
  border-color: #bee3f8;
  background: #ebf8ff;
}

.view-btn:hover {
  background: #bee3f8;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2.5rem;
  color: #718096;
}
</style>
