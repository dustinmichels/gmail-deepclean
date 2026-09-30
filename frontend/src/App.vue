<script setup lang="ts">
import { watch } from 'vue'
import { Lock, LogIn, LogOut, MailCheck, ShieldCheck, User } from 'lucide-vue'
import EmailList from './components/EmailList.vue'
import ScanDashboard from './components/ScanDashboard.vue'
import SenderTable from './components/SenderTable.vue'
import { useAuth } from './composables/useAuth'
import { useEmails } from './composables/useEmails'
import { useScan } from './composables/useScan'

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
  contactsFilter,
  contactsSearch,
  contactsSort,
  contactsLoading,
  contactsError,
  emails,
  totalEmails,
  selectedSender,
  emailsFilter,
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

// When authenticated, load initial data
watch(
  isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      fetchStatus()
      fetchStats()
      fetchContacts()
      fetchEmails()
    }
  },
  { immediate: true }
)

// When scan finishes or updates, refresh contacts and emails
watch(
  () => progress.value.totalEmails,
  () => {
    fetchStats()
    fetchContacts()
    fetchEmails()
  }
)

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
        <div class="brand-row">
          <MailCheck :size="28" class="brand-icon" />
          <h1 class="logo-title">Gmail DeepClean</h1>
          <span class="safety-badge">
            <ShieldCheck :size="13" />
            Strictly Read-Only
          </span>
        </div>
        <p class="logo-subtitle">
          Discover contacts who send you emails you've never emailed back.
        </p>
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
        Gmail DeepClean analyzes your inbox and sent emails locally to uncover communication
        breakdowns, especially
        <strong>people who email you whom you have never emailed back</strong>.
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
      <!-- Dashboard & Scan Controls (with Progress Bar) -->
      <ScanDashboard
        :progress="progress"
        :stats="stats"
        :is-scanning="isScanning"
        @start-scan="startScan"
        @pause-scan="pauseScan"
      />

      <!-- Sender Breakdown & Analysis Table -->
      <SenderTable
        :contacts="contacts"
        :total="totalContacts"
        :loading="contactsLoading"
        :error="contactsError"
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
        @retry="fetchContacts"
      />

      <!-- Detailed Emails List with Compact Previews -->
      <div id="email-list-section">
        <EmailList
          :emails="emails"
          :total="totalEmails"
          :loading="emailsLoading"
          :error="emailsError"
          :selected-sender="selectedSender"
          :search-query="emailsSearch"
          :filter-mode="emailsFilter"
          @retry="fetchEmails(false)"
          @clear-sender="clearSenderFilter"
          @update:search="emailsSearch = $event"
          @update:filter-mode="emailsFilter = $event"
          @select-sender="handleSelectSender"
          @load-more="loadMoreEmails"
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
</style>
