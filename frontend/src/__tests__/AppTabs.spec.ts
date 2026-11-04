import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import App from '../App.vue'

// Mock composables
const mockIsAuthenticated = ref(true)
const mockUserEmail = ref('user@example.com')
const mockProgress = ref({
  userId: 'user-1',
  userEmail: 'user@example.com',
  isProcessing: false,
  totalEmails: 150,
  statusMessage: 'Ready',
})
const mockStats = ref({
  totalEmailsProcessed: 150,
  totalReceived: 100,
  totalSent: 50,
  uniqueSenders: 25,
  totalSizeBytes: 102400,
  neverEmailedSenders: 10,
  neverEmailedEmails: 60,
  neverEmailedSizeBytes: 40960,
  candidateEmailsCount: 15,
  candidateSizeBytes: 30000,
  candidateSendersCount: 5,
})
const mockIsScanning = ref(false)

vi.mock('../composables/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: mockIsAuthenticated,
    userEmail: mockUserEmail,
    loading: ref(false),
    authHeaders: ref({ Authorization: 'Bearer test' }),
    startAuth: vi.fn(),
    logout: vi.fn(),
  }),
}))

vi.mock('../composables/useScan', () => ({
  useScan: () => ({
    progress: mockProgress,
    stats: mockStats,
    isScanning: mockIsScanning,
    startScan: vi.fn(),
    pauseScan: vi.fn(),
    fetchStatus: vi.fn(),
    fetchStats: vi.fn(),
  }),
}))

vi.mock('../composables/useEmails', () => ({
  useEmails: () => ({
    contacts: ref([]),
    totalContacts: ref(0),
    contactsSearch: ref(''),
    contactsSort: ref('received_count'),
    contactsLoading: ref(false),
    contactsError: ref(null),
    emails: ref([]),
    totalEmails: ref(0),
    selectedSender: ref(null),
    emailsSearch: ref(''),
    emailsLoading: ref(false),
    emailsError: ref(null),
    fetchContacts: vi.fn(),
    fetchEmails: vi.fn(),
    loadMoreEmails: vi.fn(),
    filterBySender: vi.fn(),
    clearSenderFilter: vi.fn(),
    exportContactsCSV: vi.fn(),
  }),
}))

vi.mock('../composables/useCandidates', () => ({
  useCandidates: () => ({
    senders: ref([]),
    filteredSenders: ref([]),
    loadingSenders: ref(false),
    sendersError: ref(null),
    senderSearch: ref(''),
    emails: ref([]),
    totalEmails: ref(0),
    loadingEmails: ref(false),
    emailsError: ref(null),
    selectedSender: ref(null),
    ageFilter: ref('all'),
    emailSearch: ref(''),
    selectedEmailIds: ref(new Set()),
    selectedCount: ref(0),
    selectedSize: ref(0),
    totalCandidateEmailsCount: ref(15),
    totalCandidateSizeBytes: ref(30000),
    isAllVisibleSelected: ref(false),
    fetchCandidateSenders: vi.fn(),
    fetchCandidateEmails: vi.fn(),
    loadMoreEmails: vi.fn(),
    selectSender: vi.fn(),
    toggleEmail: vi.fn(),
    toggleSelectAllVisible: vi.fn(),
    selectAllFromSender: vi.fn(),
    deselectAll: vi.fn(),
    exportSelectedCSV: vi.fn(),
    copySelectedIDs: vi.fn(),
    getRelativeAge: vi.fn(),
  }),
}))

describe('App Tabs Navigation', () => {
  it('defaults to "analyze" tab when scan has already been completed', () => {
    mockProgress.value.totalEmails = 150
    mockStats.value.totalEmailsProcessed = 150
    mockIsScanning.value = false

    const wrapper = mount(App)

    const tabs = wrapper.findAll('.tab-nav .tab-btn')
    expect(tabs).toHaveLength(3)

    // The second tab (Analyze) should be active
    expect(tabs[1].classes()).toContain('active')
    expect(tabs[1].attributes('aria-selected')).toBe('true')
  })
  it('switches tabs when tabs are clicked', async () => {
    const wrapper = mount(App)
    const tabs = wrapper.findAll('.tab-nav .tab-btn')
    // Click Scan tab
    await tabs[0].trigger('click')
    expect(tabs[0].classes()).toContain('active')

    // Click Manage tab
    await tabs[2].trigger('click')
    expect(tabs[2].classes()).toContain('active')
  })
})
