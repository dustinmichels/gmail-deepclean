<script setup lang="ts">
import { ref } from 'vue'
import {
  AlertTriangle,
  Calendar,
  Check,
  CheckSquare,
  ChevronDown,
  ChevronUp,
  Copy,
  Download,
  Filter,
  HardDrive,
  Info,
  Lock,
  Mail,
  RotateCw,
  Search,
  ShieldAlert,
  Square,
  Trash2,
  Users,
  UserX,
  X,
} from 'lucide-vue-next'
import type { CandidateSender, EmailMetadata } from '../types'

const props = defineProps<{
  senders: CandidateSender[]
  filteredSenders: CandidateSender[]
  loadingSenders: boolean
  sendersError: string | null
  senderSearch: string
  emails: EmailMetadata[]
  totalEmails: number
  loadingEmails: boolean
  emailsError: string | null
  selectedSender: string | null
  ageFilter: 'all' | '3m' | '6m' | '1y'
  emailSearch: string
  selectedEmailIds: Set<string>
  selectedCount: number
  selectedSize: number
  totalCandidateEmailsCount: number
  totalCandidateSizeBytes: number
  isAllVisibleSelected: boolean
}>()

const emit = defineEmits<{
  (e: 'update:senderSearch', val: string): void
  (e: 'update:emailSearch', val: string): void
  (e: 'update:ageFilter', val: 'all' | '3m' | '6m' | '1y'): void
  (e: 'selectSender', senderEmail: string | null): void
  (e: 'toggleEmail', email: EmailMetadata): void
  (e: 'toggleSelectAllVisible'): void
  (e: 'selectAllFromSender', senderEmail: string): void
  (e: 'deselectAll'): void
  (e: 'exportCSV'): void
  (e: 'copyIDs'): void
  (e: 'loadMore'): void
  (e: 'retrySenders'): void
  (e: 'retryEmails'): void
}>()

const expandedEmails = ref<Record<string, boolean>>({})
const copiedFeedback = ref(false)

const toggleExpand = (id: string) => {
  expandedEmails.value[id] = !expandedEmails.value[id]
}

const handleCopyIDs = async () => {
  emit('copyIDs')
  copiedFeedback.value = true
  setTimeout(() => {
    copiedFeedback.value = false
  }, 2000)
}

const formatBytes = (bytes: number): string => {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formatDate = (dateStr?: string): string => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

const getRelativeAge = (dateStr?: string): string => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diffDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24))
  if (diffDays < 30) return `${diffDays}d ago`
  const diffMonths = Math.floor(diffDays / 30)
  if (diffMonths < 12) return `${diffMonths} ${diffMonths === 1 ? 'mo' : 'mos'} old`
  const diffYears = (diffDays / 365).toFixed(1)
  return `${diffYears} yrs old`
}
</script>

<template>
  <div class="manage-container">
    <!-- Mandatory Safety & Rules Banner -->
    <div class="safety-banner">
      <div class="safety-banner-icon">
        <ShieldAlert :size="24" />
      </div>
      <div class="safety-banner-content">
        <div class="safety-banner-title">
          <strong>Candidate Evaluation Mode</strong>
          <span class="safety-chip">Strictly Read-Only — DO NOT DELETE YET</span>
        </div>
        <p class="safety-banner-desc">
          Emails shown here meet the two candidate criteria:
          <strong>1) From a sender you have never emailed back</strong>, and
          <strong>2) Received more than a month ago</strong>. Explore senders, inspect messages, and
          select candidates. No emails will be modified or deleted.
        </p>
      </div>
    </div>

    <!-- KPI Summary Row -->
    <div class="kpi-grid">
      <div class="kpi-card">
        <div class="kpi-label">
          <Mail :size="15" class="kpi-icon" />
          <span>Candidate Emails</span>
        </div>
        <div class="kpi-val">{{ totalCandidateEmailsCount.toLocaleString() }}</div>
        <div class="kpi-sub">>1 mo old from unreplied senders</div>
      </div>

      <div class="kpi-card">
        <div class="kpi-label">
          <Users :size="15" class="kpi-icon" />
          <span>Candidate Senders</span>
        </div>
        <div class="kpi-val">{{ senders.length.toLocaleString() }}</div>
        <div class="kpi-sub">Senders never emailed back</div>
      </div>

      <div class="kpi-card">
        <div class="kpi-label">
          <HardDrive :size="15" class="kpi-icon" />
          <span>Potential Reclaimable</span>
        </div>
        <div class="kpi-val">{{ formatBytes(totalCandidateSizeBytes) }}</div>
        <div class="kpi-sub">Total space of candidate messages</div>
      </div>

      <div class="kpi-card highlight-kpi" :class="{ 'has-selection': selectedCount > 0 }">
        <div class="kpi-label">
          <CheckSquare :size="15" class="kpi-icon" />
          <span>Selected for Review</span>
        </div>
        <div class="kpi-val">
          {{ selectedCount }}
          <span class="kpi-val-unit">emails</span>
        </div>
        <div class="kpi-sub">
          <template v-if="selectedCount > 0"> {{ formatBytes(selectedSize) }} selected </template>
          <template v-else> Select candidates below </template>
        </div>
      </div>
    </div>

    <!-- Active Selection Actions Toolbar -->
    <div class="selection-bar" v-if="selectedCount > 0 || emails.length > 0">
      <div class="selection-info">
        <button
          class="btn btn-outline btn-xs"
          @click="emit('toggleSelectAllVisible')"
          :title="isAllVisibleSelected ? 'Deselect visible' : 'Select all visible'"
        >
          <component :is="isAllVisibleSelected ? CheckSquare : Square" :size="14" />
          {{ isAllVisibleSelected ? 'Deselect Visible' : 'Select All Visible' }}
        </button>

        <span class="selection-status-badge">
          <strong>{{ selectedCount }}</strong> of {{ totalEmails.toLocaleString() }} selected
          <span v-if="selectedCount > 0" class="selection-size-pill">
            ({{ formatBytes(selectedSize) }})
          </span>
        </span>

        <button v-if="selectedCount > 0" class="btn btn-ghost btn-xs" @click="emit('deselectAll')">
          Clear Selection
        </button>
      </div>

      <div class="selection-actions">
        <button
          class="btn btn-outline btn-sm"
          @click="handleCopyIDs"
          :disabled="selectedCount === 0"
          title="Copy selected Gmail message IDs to clipboard"
        >
          <Check v-if="copiedFeedback" :size="14" class="text-success" />
          <Copy v-else :size="14" />
          {{ copiedFeedback ? 'Copied IDs!' : 'Copy IDs' }}
        </button>

        <button
          class="btn btn-outline btn-sm"
          @click="emit('exportCSV')"
          :disabled="totalEmails === 0"
          title="Export candidate emails as CSV spreadsheet"
        >
          <Download :size="14" />
          Export CSV
        </button>

        <!-- Safety lock: deletion disabled -->
        <button
          class="btn btn-danger-disabled btn-sm"
          disabled
          title="Deletion is strictly disabled per safety policy (DO NOT DELETE YET)"
        >
          <Lock :size="14" />
          Delete Selected (Disabled)
        </button>
      </div>
    </div>

    <!-- Age Filter & Search Bar -->
    <div class="controls-card">
      <div class="filter-pills">
        <span class="filter-pills-label">
          <Filter :size="13" />
          Age Criteria:
        </span>
        <button
          class="pill-btn"
          :class="{ active: ageFilter === 'all' }"
          @click="emit('update:ageFilter', 'all')"
        >
          >1 Month (All Candidates)
        </button>
        <button
          class="pill-btn"
          :class="{ active: ageFilter === '3m' }"
          @click="emit('update:ageFilter', '3m')"
        >
          >3 Months Old
        </button>
        <button
          class="pill-btn"
          :class="{ active: ageFilter === '6m' }"
          @click="emit('update:ageFilter', '6m')"
        >
          >6 Months Old
        </button>
        <button
          class="pill-btn"
          :class="{ active: ageFilter === '1y' }"
          @click="emit('update:ageFilter', '1y')"
        >
          >1 Year Old
        </button>
      </div>

      <div class="search-input-wrapper">
        <Search :size="14" class="search-icon" />
        <input
          type="text"
          class="search-input"
          placeholder="Search candidates by subject, snippet, or sender..."
          :value="emailSearch"
          @input="emit('update:emailSearch', ($event.target as HTMLInputElement).value)"
        />
        <button v-if="emailSearch" class="clear-search-btn" @click="emit('update:emailSearch', '')">
          <X :size="12" />
        </button>
      </div>
    </div>

    <!-- Two-Column Candidate Explorer -->
    <div class="explorer-grid">
      <!-- Left Column: Senders Sidebar -->
      <aside class="senders-panel">
        <div class="panel-header">
          <div class="panel-title-row">
            <Users :size="16" class="panel-icon" />
            <span class="panel-title">Candidate Senders</span>
            <span class="panel-count-badge">{{ filteredSenders.length }}</span>
          </div>

          <div class="sender-search-box">
            <Search :size="12" class="sender-search-icon" />
            <input
              type="text"
              class="sender-search-input"
              placeholder="Filter senders..."
              :value="senderSearch"
              @input="emit('update:senderSearch', ($event.target as HTMLInputElement).value)"
            />
            <button
              v-if="senderSearch"
              class="clear-sender-search"
              @click="emit('update:senderSearch', '')"
            >
              <X :size="11" />
            </button>
          </div>
        </div>

        <!-- Filter tag when a sender is actively selected -->
        <div v-if="selectedSender" class="active-sender-bar">
          <span class="active-sender-text">
            Filtering by: <strong>{{ selectedSender }}</strong>
          </span>
          <button
            class="clear-active-sender-btn"
            @click="emit('selectSender', null)"
            title="Show all candidate senders"
          >
            Show All
            <X :size="12" />
          </button>
        </div>

        <div v-if="loadingSenders" class="loading-state">
          <RotateCw :size="18" class="spin-icon" />
          <span>Loading senders...</span>
        </div>

        <div v-else-if="sendersError" class="error-state">
          <AlertTriangle :size="18" class="text-danger" />
          <p>{{ sendersError }}</p>
          <button class="btn btn-outline btn-xs" @click="emit('retrySenders')">Retry</button>
        </div>

        <div v-else-if="filteredSenders.length === 0" class="empty-senders">
          <p v-if="senderSearch">No senders match "{{ senderSearch }}".</p>
          <p v-else>No candidate senders found.</p>
        </div>

        <ul v-else class="senders-list">
          <li
            v-for="s in filteredSenders"
            :key="s.email"
            class="sender-item"
            :class="{ active: selectedSender === s.email }"
          >
            <div class="sender-main" @click="emit('selectSender', s.email)">
              <div class="sender-name-row">
                <span class="sender-name" :title="s.name || s.email">
                  {{ s.name || s.email.split('@')[0] }}
                </span>
                <span class="sender-count-pill">{{ s.candidateCount }}</span>
              </div>
              <div class="sender-email" :title="s.email">{{ s.email }}</div>
              <div class="sender-meta-row">
                <span class="sender-size">{{ formatBytes(s.candidateSize) }}</span>
                <span v-if="s.oldestDate" class="sender-age">
                  Oldest: {{ getRelativeAge(s.oldestDate) }}
                </span>
              </div>
            </div>

            <div class="sender-actions">
              <button
                class="btn-sender-select"
                @click.stop="emit('selectAllFromSender', s.email)"
                title="Select all visible emails from this sender"
              >
                Select
              </button>
            </div>
          </li>
        </ul>
      </aside>

      <!-- Right Column: Candidate Emails List -->
      <section class="emails-panel">
        <div class="emails-header">
          <div class="emails-header-left">
            <Mail :size="16" class="panel-icon" />
            <span class="panel-title">
              <template v-if="selectedSender">
                Candidate Emails from <strong>{{ selectedSender }}</strong>
              </template>
              <template v-else> Candidate Emails </template>
            </span>
            <span class="panel-count-badge">{{ totalEmails.toLocaleString() }}</span>
          </div>

          <div class="emails-header-right">
            <span class="criteria-tag">
              <UserX :size="12" />
              Never Emailed
            </span>
            <span class="criteria-tag">
              <Calendar :size="12" />
              >1 mo old
            </span>
          </div>
        </div>

        <!-- Emails List View -->
        <div v-if="loadingEmails && emails.length === 0" class="loading-state">
          <RotateCw :size="20" class="spin-icon" />
          <span>Loading candidate emails...</span>
        </div>

        <div v-else-if="emailsError" class="error-state">
          <AlertTriangle :size="20" class="text-danger" />
          <p>{{ emailsError }}</p>
          <button class="btn btn-outline btn-xs" @click="emit('retryEmails')">Retry</button>
        </div>

        <div v-else-if="emails.length === 0" class="empty-emails-card">
          <div class="empty-icon-circle">
            <Info :size="24" />
          </div>
          <h3>No Candidate Emails Found</h3>
          <p v-if="emailSearch">No candidate emails match your search query "{{ emailSearch }}".</p>
          <p v-else-if="selectedSender">No candidate emails found for {{ selectedSender }}.</p>
          <p v-else>
            Great news! No emails in your account match the candidate criteria (received over a
            month ago from a sender you have never emailed back).
          </p>
        </div>

        <div v-else class="emails-list">
          <div
            v-for="email in emails"
            :key="email.id"
            class="email-card"
            :class="{ 'is-selected': selectedEmailIds.has(email.id) }"
          >
            <!-- Checkbox column -->
            <div class="email-select-cell">
              <button
                class="checkbox-btn"
                @click="emit('toggleEmail', email)"
                :aria-label="selectedEmailIds.has(email.id) ? 'Deselect email' : 'Select email'"
              >
                <component
                  :is="selectedEmailIds.has(email.id) ? CheckSquare : Square"
                  :size="18"
                  :class="{ 'checkbox-checked': selectedEmailIds.has(email.id) }"
                />
              </button>
            </div>

            <!-- Email Main Information -->
            <div class="email-body" @click="toggleExpand(email.id)">
              <div class="email-top-row">
                <div class="email-sender-info">
                  <span class="email-from-name">
                    {{ email.fromName || email.from.split('@')[0] }}
                  </span>
                  <span class="email-from-addr">&lt;{{ email.from }}&gt;</span>
                </div>

                <div class="email-badges">
                  <span
                    class="badge badge-warning"
                    title="Sender has never received an email from you"
                  >
                    <UserX :size="11" />
                    Never Emailed
                  </span>
                  <span class="badge badge-age" title="Email age">
                    <Calendar :size="11" />
                    {{ getRelativeAge(email.date) }}
                  </span>
                  <span class="badge badge-size">
                    {{ formatBytes(email.sizeEstimate) }}
                  </span>
                </div>
              </div>

              <div class="email-subject-row">
                <span class="email-subject">{{ email.subject || '(no subject)' }}</span>
                <span class="email-date">{{ formatDate(email.date) }}</span>
              </div>

              <!-- Compact Snippet -->
              <p class="email-snippet" :class="{ expanded: expandedEmails[email.id] }">
                {{ email.snippet || '(empty preview)' }}
              </p>

              <!-- Expanded Details Drawer -->
              <div v-if="expandedEmails[email.id]" class="expanded-drawer" @click.stop>
                <div class="detail-row">
                  <span class="detail-label">Message ID:</span>
                  <code class="detail-code">{{ email.id }}</code>
                </div>
                <div class="detail-row" v-if="email.to && email.to.length">
                  <span class="detail-label">To:</span>
                  <span>{{ email.to.join(', ') }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Exact Date:</span>
                  <span>{{ new Date(email.date).toLocaleString() }}</span>
                </div>
                <div class="detail-row" v-if="email.labelIds && email.labelIds.length">
                  <span class="detail-label">Labels:</span>
                  <div class="label-pills">
                    <span v-for="lbl in email.labelIds" :key="lbl" class="label-pill">
                      {{ lbl }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Expand Toggle Button -->
            <button
              class="expand-btn"
              @click="toggleExpand(email.id)"
              :title="expandedEmails[email.id] ? 'Collapse' : 'Expand'"
            >
              <component :is="expandedEmails[email.id] ? ChevronUp : ChevronDown" :size="15" />
            </button>
          </div>

          <!-- Pagination: Load More Button -->
          <div class="load-more-row" v-if="emails.length < totalEmails">
            <button
              class="btn btn-outline btn-sm"
              @click="emit('loadMore')"
              :disabled="loadingEmails"
            >
              <RotateCw v-if="loadingEmails" :size="14" class="spin-icon" />
              <span>
                {{
                  loadingEmails
                    ? 'Loading more...'
                    : `Load More (${emails.length} of ${totalEmails})`
                }}
              </span>
            </button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.manage-container {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  width: 100%;
}

/* Safety & Rule Banner */
.safety-banner {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: var(--radius-md, 10px);
  padding: 0.9rem 1.15rem;
}

.safety-banner-icon {
  color: #b45309;
  flex-shrink: 0;
  margin-top: 0.15rem;
}

.safety-banner-content {
  flex: 1;
}

.safety-banner-title {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
  font-size: 0.92rem;
  color: #78350f;
  margin-bottom: 0.25rem;
}

.safety-chip {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fcd34d;
  font-size: 0.72rem;
  font-weight: 700;
  padding: 0.12rem 0.5rem;
  border-radius: 999px;
  letter-spacing: 0.02em;
}

.safety-banner-desc {
  font-size: 0.82rem;
  color: #92400e;
  line-height: 1.45;
  margin: 0;
}

/* KPI Summary Cards */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.85rem;
}

.kpi-card {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 0.85rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.kpi-label {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--color-text-muted, #64748b);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.kpi-icon {
  color: var(--color-primary, #2563eb);
}

.kpi-val {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--color-text-main, #0f172a);
  line-height: 1.2;
}

.kpi-val-unit {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-muted, #64748b);
  margin-left: 0.2rem;
}

.kpi-sub {
  font-size: 0.75rem;
  color: var(--color-text-muted, #64748b);
}

.highlight-kpi {
  background: #f8fafc;
  border-color: #cbd5e1;
}

.highlight-kpi.has-selection {
  background: #eff6ff;
  border-color: #bfdbfe;
}

.highlight-kpi.has-selection .kpi-val {
  color: var(--color-primary, #2563eb);
}

/* Selection Action Bar */
.selection-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
  background: #ffffff;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  padding: 0.6rem 0.9rem;
  box-shadow: var(--shadow-sm);
}

.selection-info {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.selection-status-badge {
  font-size: 0.82rem;
  color: var(--color-text-main, #0f172a);
}

.selection-size-pill {
  color: var(--color-text-muted, #64748b);
  font-size: 0.78rem;
}

.selection-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.btn-danger-disabled {
  background: #fee2e2;
  color: #991b1b;
  border-color: #fecaca;
  opacity: 0.65;
  cursor: not-allowed;
}

.btn-ghost {
  background: transparent;
  border: none;
  color: var(--color-primary, #2563eb);
  cursor: pointer;
  padding: 0.2rem 0.4rem;
  font-size: 0.78rem;
  font-weight: 600;
}

.btn-ghost:hover {
  text-decoration: underline;
}

.btn-xs {
  padding: 0.25rem 0.55rem;
  font-size: 0.75rem;
}

/* Controls Card */
.controls-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  padding: 0.6rem 0.9rem;
}

.filter-pills {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.filter-pills-label {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--color-text-muted, #64748b);
  margin-right: 0.2rem;
}

.pill-btn {
  background: var(--color-bg, #f8fafc);
  border: 1px solid var(--color-border, #e2e8f0);
  color: var(--color-text-muted, #475569);
  padding: 0.28rem 0.65rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.pill-btn:hover {
  background: var(--color-surface-hover, #f1f5f9);
  color: var(--color-text-main, #0f172a);
}

.pill-btn.active {
  background: var(--color-primary-light, #eff6ff);
  color: var(--color-primary, #2563eb);
  border-color: var(--color-primary-border, #bfdbfe);
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  min-width: 260px;
  flex: 1;
  max-width: 420px;
}

.search-icon {
  position: absolute;
  left: 0.65rem;
  color: var(--color-text-faint, #94a3b8);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.35rem 1.8rem 0.35rem 2rem;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  font-size: 0.82rem;
  outline: none;
  background: var(--color-surface, #ffffff);
}

.search-input:focus {
  border-color: var(--color-primary, #2563eb);
}

.clear-search-btn {
  position: absolute;
  right: 0.5rem;
  background: none;
  border: none;
  color: var(--color-text-faint, #94a3b8);
  cursor: pointer;
  padding: 0.15rem;
}

/* Two-Column Explorer Layout */
.explorer-grid {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 1rem;
  align-items: start;
}

@media (max-width: 900px) {
  .explorer-grid {
    grid-template-columns: 1fr;
  }
}

/* Senders Panel */
.senders-panel {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  max-height: 800px;
}

.panel-header {
  padding: 0.75rem 0.85rem;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: #f8fafc;
}

.panel-title-row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.panel-icon {
  color: var(--color-primary, #2563eb);
}

.panel-title {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
}

.panel-count-badge {
  background: var(--color-border, #e2e8f0);
  color: var(--color-text-muted, #475569);
  font-size: 0.72rem;
  font-weight: 700;
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  margin-left: auto;
}

.sender-search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.sender-search-icon {
  position: absolute;
  left: 0.5rem;
  color: var(--color-text-faint, #94a3b8);
}

.sender-search-input {
  width: 100%;
  padding: 0.28rem 1.6rem 0.28rem 1.7rem;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 4px;
  font-size: 0.78rem;
  background: #ffffff;
}

.clear-sender-search {
  position: absolute;
  right: 0.4rem;
  background: none;
  border: none;
  color: var(--color-text-faint, #94a3b8);
  cursor: pointer;
}

.active-sender-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--color-primary-light, #eff6ff);
  border-bottom: 1px solid var(--color-primary-border, #bfdbfe);
  padding: 0.4rem 0.75rem;
  font-size: 0.75rem;
  color: var(--color-primary, #2563eb);
}

.active-sender-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clear-active-sender-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: none;
  border: none;
  color: var(--color-primary, #2563eb);
  font-size: 0.72rem;
  font-weight: 700;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
}

.senders-list {
  list-style: none;
  overflow-y: auto;
  margin: 0;
  padding: 0;
  max-height: 700px;
}

.sender-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 0.85rem;
  border-bottom: 1px solid var(--color-border-subtle, #f1f5f9);
  transition: background 0.15s ease;
  gap: 0.5rem;
}

.sender-item:hover {
  background: var(--color-surface-hover, #f8fafc);
}

.sender-item.active {
  background: var(--color-primary-light, #eff6ff);
  border-left: 3px solid var(--color-primary, #2563eb);
}

.sender-main {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}

.sender-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.4rem;
}

.sender-name {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sender-count-pill {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.05rem 0.4rem;
  border-radius: 999px;
  flex-shrink: 0;
}

.sender-email {
  font-size: 0.72rem;
  color: var(--color-text-muted, #64748b);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sender-meta-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-top: 0.15rem;
  font-size: 0.7rem;
  color: var(--color-text-faint, #94a3b8);
}

.sender-size {
  font-weight: 600;
  color: var(--color-text-muted, #475569);
}

.btn-sender-select {
  background: none;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 4px;
  padding: 0.2rem 0.45rem;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--color-text-muted, #475569);
  cursor: pointer;
}

.btn-sender-select:hover {
  background: var(--color-surface, #ffffff);
  color: var(--color-primary, #2563eb);
  border-color: var(--color-primary-border, #bfdbfe);
}

.empty-senders {
  padding: 2rem 1rem;
  text-align: center;
  font-size: 0.8rem;
  color: var(--color-text-muted, #64748b);
}

/* Emails Panel */
.emails-panel {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-md, 10px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.emails-header {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  background: #f8fafc;
}

.emails-header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.emails-header-right {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.criteria-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.72rem;
  font-weight: 600;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  background: #f1f5f9;
  color: var(--color-text-muted, #475569);
  border: 1px solid var(--color-border, #e2e8f0);
}

/* Emails List */
.emails-list {
  display: flex;
  flex-direction: column;
}

.email-card {
  display: flex;
  align-items: flex-start;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--color-border-subtle, #f1f5f9);
  gap: 0.75rem;
  transition: all 0.15s ease;
  position: relative;
}

.email-card:hover {
  background: #fcfdfe;
}

.email-card.is-selected {
  background: #eff6ff;
  border-left: 3px solid var(--color-primary, #2563eb);
}

.email-select-cell {
  flex-shrink: 0;
  margin-top: 0.15rem;
}

.checkbox-btn {
  background: none;
  border: none;
  color: var(--color-text-faint, #94a3b8);
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.checkbox-btn:hover {
  color: var(--color-primary, #2563eb);
}

.checkbox-checked {
  color: var(--color-primary, #2563eb);
}

.email-body {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}

.email-top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 0.2rem;
}

.email-sender-info {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.email-from-name {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
}

.email-from-addr {
  font-size: 0.75rem;
  color: var(--color-text-muted, #64748b);
}

.email-badges {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.12rem 0.45rem;
  border-radius: 4px;
}

.badge-warning {
  background: #fff7ed;
  color: #9a3412;
  border: 1px solid #fed7aa;
}

.badge-age {
  background: #eff6ff;
  color: #1e40af;
  border: 1px solid #bfdbfe;
}

.badge-size {
  background: #f1f5f9;
  color: #475569;
  border: 1px solid #e2e8f0;
}

.email-subject-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.email-subject {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-main, #0f172a);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.email-date {
  font-size: 0.72rem;
  color: var(--color-text-faint, #94a3b8);
  flex-shrink: 0;
}

.email-snippet {
  font-size: 0.78rem;
  color: var(--color-text-muted, #64748b);
  line-height: 1.4;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.email-snippet.expanded {
  -webkit-line-clamp: unset;
  display: block;
}

.expand-btn {
  background: none;
  border: none;
  color: var(--color-text-faint, #94a3b8);
  cursor: pointer;
  padding: 0.2rem;
  flex-shrink: 0;
  margin-top: 0.15rem;
}

.expand-btn:hover {
  color: var(--color-text-main, #0f172a);
}

/* Expanded Drawer */
.expanded-drawer {
  margin-top: 0.6rem;
  padding: 0.6rem 0.75rem;
  background: #f8fafc;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  font-size: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.detail-row {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.detail-label {
  font-weight: 700;
  color: var(--color-text-muted, #475569);
  min-width: 80px;
}

.detail-code {
  font-family: monospace;
  font-size: 0.72rem;
  background: #ffffff;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  border: 1px solid #e2e8f0;
}

.label-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.label-pill {
  background: #e2e8f0;
  color: #334155;
  font-size: 0.68rem;
  padding: 0.05rem 0.35rem;
  border-radius: 3px;
  font-weight: 600;
}

.load-more-row {
  padding: 1rem;
  display: flex;
  justify-content: center;
  border-top: 1px solid var(--color-border, #e2e8f0);
  background: #f8fafc;
}

/* Empty & Loading States */
.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 3rem 1.5rem;
  color: var(--color-text-muted, #64748b);
  font-size: 0.85rem;
  text-align: center;
}

.spin-icon {
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

.empty-emails-card {
  text-align: center;
  padding: 3.5rem 1.5rem;
  color: var(--color-text-muted, #64748b);
}

.empty-icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: #f1f5f9;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 0.75rem;
}

.empty-emails-card h3 {
  font-size: 1rem;
  font-weight: 700;
  color: var(--color-text-main, #0f172a);
  margin-bottom: 0.35rem;
}

.empty-emails-card p {
  font-size: 0.82rem;
  max-width: 440px;
  margin: 0 auto;
  line-height: 1.5;
}

.text-success {
  color: #166534;
}

.text-danger {
  color: #b91c1c;
}
</style>
