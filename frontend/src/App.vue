<script setup lang="ts">
import { watch } from 'vue'
import EmailList from './components/EmailList.vue'
import ScanDashboard from './components/ScanDashboard.vue'
import SenderTable from './components/SenderTable.vue'
import { useAuth } from './composables/useAuth'
import { useEmails } from './composables/useEmails'
import { useScan } from './composables/useScan'

const { isAuthenticated, userEmail, loading: authLoading, authHeaders, startAuth, logout } = useAuth()

const { progress, stats, isScanning, startScan, pauseScan, fetchStatus, fetchStats } = useScan(
  () => authHeaders.value
)

const {
  contacts,
  totalContacts,
  contactsFilter,
  contactsSearch,
  contactsSort,
  contactsLoading,
  emails,
  totalEmails,
  selectedSender,
  emailsSearch,
  emailsLoading,
  fetchContacts,
  fetchEmails,
  loadMoreEmails,
  filterBySender,
  clearSenderFilter,
  exportContactsCSV,
} = useEmails(() => authHeaders.value)

// When authenticated, load initial data
watch(isAuthenticated, (authenticated) => {
  if (authenticated) {
    fetchStatus()
    fetchStats()
    fetchContacts()
    fetchEmails()
  }
}, { immediate: true })

// When scan finishes or updates, refresh contacts and emails
watch(
  () => progress.value.totalEmails,
  () => {
    fetchStats()
    fetchContacts()
    fetchEmails()
  }
)

const handleSelectTab = (tab: string) => {
  contactsFilter.value = tab
}

const handleSelectSender = (senderEmail: string) => {
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
        <h1 class="logo-title">Gmail DeepClean</h1>
        <p class="logo-subtitle">
          Identify contacts who send you emails you've never emailed back — strictly read-only.
        </p>
      </div>

      <div class="header-user" v-if="isAuthenticated">
        <span class="user-badge">{{ userEmail }}</span>
        <button @click="logout" class="btn btn-outline btn-sm">Logout</button>
      </div>
    </header>

    <!-- Auth Callout for Unauthenticated -->
    <div v-if="!isAuthenticated" class="auth-card">
      <div class="auth-icon">🔒</div>
      <h2>Connect Your Gmail Account</h2>
      <p>
        Gmail DeepClean analyzes your inbox and sent emails locally to uncover communication
        breakdowns, especially <strong>people who email you whom you have never emailed</strong>.
      </p>
      <div class="safety-note">
        🛡 <strong>Read-Only Guarantee:</strong> This application only reads email headers and
        metadata. It will <em>never</em> send, delete, or alter any emails in your account.
      </div>
      <button @click="startAuth" class="btn btn-primary btn-lg" :disabled="authLoading">
        {{ authLoading ? 'Checking session...' : 'Connect to Gmail' }}
      </button>
    </div>

    <!-- Main Dashboard When Authenticated -->
    <main v-else class="main-content">
      <!-- Dashboard & Scan Controls -->
      <ScanDashboard
        :progress="progress"
        :stats="stats"
        :is-scanning="isScanning"
        @start-scan="startScan"
        @pause-scan="pauseScan"
        @select-tab="handleSelectTab"
      />

      <!-- Sender Breakdown & Analysis Table -->
      <SenderTable
        :contacts="contacts"
        :total="totalContacts"
        :loading="contactsLoading"
        :active-filter="contactsFilter"
        :search-query="contactsSearch"
        :sort-field="contactsSort"
        sort-order="desc"
        :never-emailed-count="stats.neverEmailedCount"
        @update:filter="contactsFilter = $event"
        @update:search="contactsSearch = $event"
        @update:sort="contactsSort = $event"
        @select-sender="handleSelectSender"
        @export-csv="exportContactsCSV"
      />

      <!-- Detailed Emails List -->
      <div id="email-list-section">
        <EmailList
          :emails="emails"
          :total="totalEmails"
          :loading="emailsLoading"
          :selected-sender="selectedSender"
          :search-query="emailsSearch"
          @clear-sender="clearSenderFilter"
          @update:search="emailsSearch = $event"
          @load-more="loadMoreEmails"
        />
      </div>
    </main>
  </div>
</template>

<style scoped>
.app-container {
  max-width: 1140px;
  margin: 0 auto;
  padding: 1.5rem 1rem 3rem;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  color: #2d3748;
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  padding-bottom: 1.25rem;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid #e2e8f0;
}

.logo-title {
  font-size: 1.75rem;
  font-weight: 800;
  color: #2b6cb0;
  margin: 0 0 0.25rem 0;
  letter-spacing: -0.02em;
}

.logo-subtitle {
  margin: 0;
  font-size: 0.9rem;
  color: #718096;
}

.header-user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-badge {
  background: #edf2f7;
  color: #4a5568;
  font-size: 0.85rem;
  font-weight: 600;
  padding: 0.35rem 0.75rem;
  border-radius: 20px;
}

.auth-card {
  text-align: center;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 3rem 1.5rem;
  max-width: 580px;
  margin: 3rem auto;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
}

.auth-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.auth-card h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 0.75rem;
  color: #1a202c;
}

.auth-card p {
  color: #4a5568;
  line-height: 1.6;
  font-size: 1rem;
  margin-bottom: 1.5rem;
}

.safety-note {
  background: #ebf8ff;
  border: 1px solid #bee3f8;
  border-radius: 8px;
  padding: 0.85rem 1rem;
  font-size: 0.85rem;
  color: #2b6cb0;
  line-height: 1.5;
  margin-bottom: 2rem;
  text-align: left;
}

.main-content {
  display: flex;
  flex-direction: column;
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

.btn-primary:hover:not(:disabled) {
  background: #2b6cb0;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-outline {
  background: transparent;
  border: 1px solid #cbd5e0;
  color: #4a5568;
}

.btn-outline:hover {
  background: #edf2f7;
}

.btn-sm {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}

.btn-lg {
  padding: 0.75rem 2rem;
  font-size: 1rem;
}
</style>
