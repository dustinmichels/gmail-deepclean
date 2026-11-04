<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  Activity,
  BarChart3,
  ListChecks,
  Lock,
  LogIn,
  LogOut,
  MailCheck,
  ShieldCheck,
  User,
} from 'lucide-vue-next'
import EmailList from './components/EmailList.vue'
import ManageCandidates from './components/ManageCandidates.vue'
import ScanDashboard from './components/ScanDashboard.vue'
import SenderTable from './components/SenderTable.vue'
import { useAuth } from './composables/useAuth'
import { useCandidates } from './composables/useCandidates'
import { useEmails } from './composables/useEmails'
import { useScan } from './composables/useScan'

type TabType = 'scan' | 'analyze' | 'manage'
const currentTab = ref<TabType>('scan')
const initialTabDetermined = ref(false)

const {
  isAuthenticated,
  userEmail,
  loading: authLoading,
  authHeaders,
  startAuth,
  logout,
} = useAuth()

const { progress, stats, isScanning, startScan, pauseScan, fetchStatus, fetchStats } = useScan(
  () => authHeaders.value
)

const {
  contacts,
  totalContacts,
  contactsSearch,
  contactsSort,
  contactsLoading,
  contactsError,
  emails,
  totalEmails,
  selectedSender,
  emailsSearch,
  emailsLoading,
  emailsError,
  fetchContacts,
  fetchEmails,
  loadMoreEmails,
  filterBySender,
  clearSenderFilter,
  exportContactsCSV,
} = useEmails(() => authHeaders.value)

const {
  senders: candidateSenders,
  filteredSenders: filteredCandidateSenders,
  loadingSenders: loadingCandidateSenders,
  sendersError: candidateSendersError,
  senderSearch: candidateSenderSearch,
  emails: candidateEmails,
  totalEmails: totalCandidateEmails,
  loadingEmails: loadingCandidateEmails,
  emailsError: candidateEmailsError,
  selectedSender: selectedCandidateSender,
  ageFilter: candidateAgeFilter,
  emailSearch: candidateEmailSearch,
  selectedEmailIds,
  selectedCount: candidateSelectedCount,
  selectedSize: candidateSelectedSize,
  totalCandidateEmailsCount,
  totalCandidateSizeBytes,
  isAllVisibleSelected: isAllCandidateVisibleSelected,
  fetchCandidateSenders,
  fetchCandidateEmails,
  loadMoreEmails: loadMoreCandidateEmails,
  selectSender: selectCandidateSender,
  toggleEmail: toggleCandidateEmail,
  toggleSelectAllVisible: toggleSelectAllCandidateVisible,
  selectAllFromSender: selectAllCandidatesFromSender,
  deselectAll: deselectAllCandidates,
  exportSelectedCSV: exportCandidatesCSV,
  copySelectedIDs: copyCandidateIDs,
} = useCandidates(() => authHeaders.value)

// When authenticated, load initial data
watch(
  isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      fetchStatus()
      fetchStats()
      fetchContacts()
      fetchEmails()
      fetchCandidateSenders()
      fetchCandidateEmails()
    }
  },
  { immediate: true }
)

// Determine default tab on load: if a scan has already been completed, "analyze" is the default tab.
watch(
  [() => progress.value.totalEmails, () => stats.value.totalEmailsProcessed, isScanning],
  ([totalScanned, processed, scanning]) => {
    if (!initialTabDetermined.value && !scanning) {
      if (totalScanned > 0 || processed > 0) {
        currentTab.value = 'analyze'
        initialTabDetermined.value = true
      }
    }
  },
  { immediate: true }
)

// When an active scan completes, automatically switch to the "analyze" tab
watch(isScanning, (scanning, wasScanning) => {
  if (wasScanning && !scanning && progress.value.totalEmails > 0) {
    currentTab.value = 'analyze'
  }
})

// When scan finishes or updates, refresh all data
watch(
  () => progress.value.totalEmails,
  () => {
    fetchStats()
    fetchContacts()
    fetchEmails()
    fetchCandidateSenders()
    fetchCandidateEmails()
  }
)

const handleSelectSender = (senderEmail: string) => {
  currentTab.value = 'analyze'
  filterBySender(senderEmail)
  // Scroll to email list smoothly
  const el = document.getElementById('email-list-section')
  if (el) {
    el.scrollIntoView({ behavior: 'smooth' })
  }
}
</script>

<template>
  <div class="app-container">
    <!-- Navigation Header -->
    <header class="app-header">
      <div class="header-brand">
        <div class="brand-row">
          <MailCheck :size="28" class="brand-icon" />
          <h1 class="logo-title">Gmail DeepClean</h1>
          <span class="safety-badge">
            <ShieldCheck :size="13" />
            Strictly Read-Only
          </span>
        </div>
        <p class="logo-subtitle">Clean up mail from people you've never emailed.</p>
      </div>

      <div class="header-user" v-if="isAuthenticated">
        <span class="user-badge">
          <User :size="14" />
          {{ userEmail }}
        </span>
        <button @click="logout" class="btn btn-outline btn-sm">
          <LogOut :size="14" />
          Logout
        </button>
      </div>
    </header>

    <!-- Auth Callout for Unauthenticated -->
    <div v-if="!isAuthenticated" class="auth-card">
      <div class="auth-icon-wrapper">
        <Lock :size="32" class="auth-lock-icon" />
      </div>
      <h2>Connect Your Gmail Account</h2>
      <p>
        Gmail DeepClean analyzes your inbox and sent emails locally to find mail from
        <strong>people you have never emailed</strong>. Anyone you have emailed is treated as a real
        contact and left alone.
      </p>
      <div class="safety-note">
        <ShieldCheck :size="18" class="safety-note-icon" />
        <div class="safety-note-text">
          <strong>Read-Only Guarantee:</strong> This application strictly reads email headers and
          snippets. It will <em>never</em> send, delete, or alter any emails in your account.
        </div>
      </div>
      <button @click="startAuth" class="btn btn-primary btn-lg" :disabled="authLoading">
        <LogIn :size="18" />
        {{ authLoading ? 'Checking session...' : 'Connect to Gmail' }}
      </button>
    </div>

    <!-- Main Dashboard When Authenticated -->
    <main v-else class="main-content">
      <!-- Tab Navigation -->
      <nav class="tab-nav" role="tablist" aria-label="Main Section Tabs">
        <button
          class="tab-btn"
          :class="{ active: currentTab === 'scan' }"
          @click="currentTab = 'scan'"
          role="tab"
          :aria-selected="currentTab === 'scan'"
        >
          <Activity :size="15" class="tab-icon" />
          <span>Scan</span>
          <span v-if="isScanning" class="tab-badge tab-badge-live">Scanning...</span>
        </button>

        <button
          class="tab-btn"
          :class="{ active: currentTab === 'analyze' }"
          @click="currentTab = 'analyze'"
          role="tab"
          :aria-selected="currentTab === 'analyze'"
        >
          <BarChart3 :size="15" class="tab-icon" />
          <span>Analyze</span>
          <span v-if="stats.neverEmailedSenders > 0" class="tab-badge">
            {{ stats.neverEmailedSenders.toLocaleString() }}
          </span>
        </button>

        <button
          class="tab-btn"
          :class="{ active: currentTab === 'manage' }"
          @click="currentTab = 'manage'"
          role="tab"
          :aria-selected="currentTab === 'manage'"
        >
          <ListChecks :size="15" class="tab-icon" />
          <span>Manage</span>
          <span v-if="totalCandidateEmailsCount > 0" class="tab-badge tab-badge-candidate">
            {{ totalCandidateEmailsCount.toLocaleString() }}
          </span>
        </button>
      </nav>

      <!-- Tab 1: Scan Controls & Overall Stats -->
      <div v-show="currentTab === 'scan'" class="tab-pane">
        <ScanDashboard
          :progress="progress"
          :stats="stats"
          :is-scanning="isScanning"
          @start-scan="startScan"
          @pause-scan="pauseScan"
          @go-to-analyze="currentTab = 'analyze'"
        />
      </div>

      <!-- Tab 2: Analyze Breakdown & Emails -->
      <div v-show="currentTab === 'analyze'" class="tab-pane">
        <SenderTable
          :contacts="contacts"
          :total="totalContacts"
          :loading="contactsLoading"
          :error="contactsError"
          :search-query="contactsSearch"
          :sort-field="contactsSort"
          @update:search="contactsSearch = $event"
          @update:sort="contactsSort = $event"
          @select-sender="handleSelectSender"
          @export-csv="exportContactsCSV"
          @retry="fetchContacts"
        />

        <div id="email-list-section">
          <EmailList
            :emails="emails"
            :total="totalEmails"
            :loading="emailsLoading"
            :error="emailsError"
            :selected-sender="selectedSender"
            :search-query="emailsSearch"
            @retry="fetchEmails(false)"
            @clear-sender="clearSenderFilter"
            @update:search="emailsSearch = $event"
            @select-sender="handleSelectSender"
            @load-more="loadMoreEmails"
          />
        </div>
      </div>

      <!-- Tab 3: Manage Candidates for Deletion -->
      <div v-show="currentTab === 'manage'" class="tab-pane">
        <ManageCandidates
          :senders="candidateSenders"
          :filtered-senders="filteredCandidateSenders"
          :loading-senders="loadingCandidateSenders"
          :senders-error="candidateSendersError"
          :sender-search="candidateSenderSearch"
          :emails="candidateEmails"
          :total-emails="totalCandidateEmails"
          :loading-emails="loadingCandidateEmails"
          :emails-error="candidateEmailsError"
          :selected-sender="selectedCandidateSender"
          :age-filter="candidateAgeFilter"
          :email-search="candidateEmailSearch"
          :selected-email-ids="selectedEmailIds"
          :selected-count="candidateSelectedCount"
          :selected-size="candidateSelectedSize"
          :total-candidate-emails-count="totalCandidateEmailsCount"
          :total-candidate-size-bytes="totalCandidateSizeBytes"
          :is-all-visible-selected="isAllCandidateVisibleSelected"
          @update:sender-search="candidateSenderSearch = $event"
          @update:email-search="candidateEmailSearch = $event"
          @update:age-filter="candidateAgeFilter = $event"
          @select-sender="selectCandidateSender"
          @toggle-email="toggleCandidateEmail"
          @toggle-select-all-visible="toggleSelectAllCandidateVisible"
          @select-all-from-sender="selectAllCandidatesFromSender"
          @deselect-all="deselectAllCandidates"
          @export-c-s-v="exportCandidatesCSV"
          @copy-i-ds="copyCandidateIDs"
          @load-more="loadMoreCandidateEmails"
          @retry-senders="fetchCandidateSenders"
          @retry-emails="fetchCandidateEmails(false)"
        />
      </div>
    </main>
  </div>
</template>

<style scoped>
.app-container {
  width: 100%;
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  padding-bottom: 1.25rem;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
}

.brand-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.logo-title {
  font-size: 1.6rem;
  font-weight: 800;
  color: var(--color-primary, #2563eb);
  letter-spacing: -0.02em;
  margin: 0;
}

.brand-icon {
  color: var(--color-primary, #2563eb);
  flex-shrink: 0;
}

.safety-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: #f0fdf4;
  color: #166534;
  border: 1px solid #bbf7d0;
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  letter-spacing: 0.02em;
}

.logo-subtitle {
  margin: 0.2rem 0 0 0;
  font-size: 0.875rem;
  color: var(--color-text-muted, #64748b);
}

.header-user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--color-surface-hover, #f1f5f9);
  color: var(--color-text-main, #0f172a);
  border: 1px solid var(--color-border, #e2e8f0);
  font-size: 0.85rem;
  font-weight: 600;
  padding: 0.35rem 0.75rem;
  border-radius: 20px;
}

/* Auth Card */
.auth-card {
  text-align: center;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-lg, 14px);
  padding: 3rem 2rem;
  max-width: 580px;
  margin: 3rem auto;
  box-shadow: var(--shadow-md);
}

.auth-icon-wrapper {
  width: 64px;
  height: 64px;
  margin: 0 auto 1.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-light, #eff6ff);
  color: var(--color-primary, #2563eb);
  border-radius: 50%;
}

.auth-lock-icon {
  color: var(--color-primary, #2563eb);
}

.auth-card h2 {
  font-size: 1.5rem;
  font-weight: 800;
  margin-bottom: 0.75rem;
  color: var(--color-text-main, #0f172a);
}

.auth-card p {
  color: var(--color-text-muted, #475569);
  line-height: 1.6;
  font-size: 0.95rem;
  margin-bottom: 1.5rem;
}

.safety-note {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: var(--radius-sm, 6px);
  padding: 0.85rem 1rem;
  font-size: 0.85rem;
  color: #166534;
  line-height: 1.5;
  margin-bottom: 2rem;
  text-align: left;
}

.safety-note-icon {
  flex-shrink: 0;
  margin-top: 0.15rem;
}

.safety-note-text {
  flex: 1;
}

.main-content {
  display: flex;
  flex-direction: column;
}

/* Tab Navigation Bar */
.tab-nav {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
  padding-bottom: 0.6rem;
  flex-wrap: wrap;
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.9rem;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: var(--radius-sm, 6px);
  color: var(--color-text-muted, #475569);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  background: var(--color-surface-hover, #f1f5f9);
  color: var(--color-text-main, #0f172a);
}

.tab-btn.active {
  background: var(--color-primary-light, #eff6ff);
  color: var(--color-primary, #2563eb);
  border-color: var(--color-primary-border, #bfdbfe);
  box-shadow: 0 1px 2px rgba(37, 99, 235, 0.08);
}

.tab-icon {
  flex-shrink: 0;
}

.tab-badge {
  background: var(--color-border, #e2e8f0);
  color: var(--color-text-muted, #475569);
  font-size: 0.72rem;
  font-weight: 700;
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  margin-left: 0.2rem;
}

.tab-btn.active .tab-badge {
  background: #dbeafe;
  color: #1d4ed8;
}

.tab-badge-live {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
  animation: pulse-badge 1.5s ease-in-out infinite;
}

@keyframes pulse-badge {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

.tab-badge-candidate {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.tab-btn.active .tab-badge-candidate {
  background: #fecaca;
  color: #7f1d1d;
}

.tab-pane {
  width: 100%;
}
</style>
