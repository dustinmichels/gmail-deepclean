import { computed, ref, watch } from 'vue'
import type { CandidateSender, EmailMetadata, EmailsResponse } from '../types'

export function useCandidates(authHeaders: () => Record<string, string>) {
  // Senders list state
  const senders = ref<CandidateSender[]>([])
  const loadingSenders = ref(false)
  const sendersError = ref<string | null>(null)
  const senderSearch = ref('')

  // Emails list state
  const emails = ref<EmailMetadata[]>([])
  const totalEmails = ref(0)
  const loadingEmails = ref(false)
  const emailsError = ref<string | null>(null)
  const emailsOffset = ref(0)
  const emailsLimit = 50

  // Filter & Search state
  const selectedSender = ref<string | null>(null)
  const ageFilter = ref<'all' | '3m' | '6m' | '1y'>('all')
  const emailSearch = ref('')

  // Multi-selection state
  const selectedEmailIds = ref<Set<string>>(new Set())
  // Cache of selected email metadata so sizes and export work across pagination
  const selectedEmailsMap = ref<Map<string, EmailMetadata>>(new Map())

  // Computed filtered senders
  const filteredSenders = computed(() => {
    const q = senderSearch.value.trim().toLowerCase()
    if (!q) return senders.value
    return senders.value.filter(
      (s) => s.email.toLowerCase().includes(q) || (s.name && s.name.toLowerCase().includes(q))
    )
  })

  // Selected count & size
  const selectedCount = computed(() => selectedEmailIds.value.size)
  const selectedSize = computed(() => {
    let bytes = 0
    for (const em of selectedEmailsMap.value.values()) {
      if (selectedEmailIds.value.has(em.id)) {
        bytes += em.sizeEstimate || 0
      }
    }
    return bytes
  })

  // Total candidate stats aggregated across all candidate senders
  const totalCandidateEmailsCount = computed(() =>
    senders.value.reduce((acc, s) => acc + s.candidateCount, 0)
  )

  const totalCandidateSizeBytes = computed(() =>
    senders.value.reduce((acc, s) => acc + s.candidateSize, 0)
  )

  const isAllVisibleSelected = computed(() => {
    if (emails.value.length === 0) return false
    return emails.value.every((em) => selectedEmailIds.value.has(em.id))
  })

  // Fetch candidate senders
  const fetchCandidateSenders = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    loadingSenders.value = true
    sendersError.value = null
    try {
      const res = await fetch('/api/inbox/candidate-senders', { headers })
      if (res.ok) {
        const data: CandidateSender[] = await res.json()
        senders.value = data || []
      } else {
        const errText = await res.text()
        sendersError.value = errText || `Failed to fetch candidate senders (HTTP ${res.status})`
      }
    } catch (e: unknown) {
      sendersError.value =
        e instanceof Error ? e.message : 'Network error fetching candidate senders'
    } finally {
      loadingSenders.value = false
    }
  }

  // Determine filter query string based on ageFilter
  const getFilterParam = () => {
    switch (ageFilter.value) {
      case '3m':
        return 'candidates_3m'
      case '6m':
        return 'candidates_6m'
      case '1y':
        return 'candidates_1y'
      default:
        return 'candidates'
    }
  }

  // Fetch candidate emails
  const fetchCandidateEmails = async (append = false) => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    loadingEmails.value = true
    emailsError.value = null
    try {
      if (!append) {
        emailsOffset.value = 0
      }

      const params = new URLSearchParams({
        filter: getFilterParam(),
        limit: String(emailsLimit),
        offset: String(emailsOffset.value),
      })

      if (selectedSender.value) {
        params.set('sender', selectedSender.value)
      }
      if (emailSearch.value.trim()) {
        params.set('search', emailSearch.value.trim())
      }

      const res = await fetch(`/api/emails?${params.toString()}`, { headers })
      if (res.ok) {
        const data: EmailsResponse = await res.json()
        const newEmails = data.emails || []
        if (append) {
          emails.value = [...emails.value, ...newEmails]
        } else {
          emails.value = newEmails
        }
        totalEmails.value = data.total || 0

        // Cache metadata for currently loaded items
        for (const em of newEmails) {
          if (selectedEmailIds.value.has(em.id)) {
            selectedEmailsMap.value.set(em.id, em)
          }
        }
      } else {
        const errText = await res.text()
        emailsError.value = errText || `Failed to fetch candidate emails (HTTP ${res.status})`
      }
    } catch (e: unknown) {
      emailsError.value = e instanceof Error ? e.message : 'Network error fetching candidate emails'
    } finally {
      loadingEmails.value = false
    }
  }

  const loadMoreEmails = async () => {
    if (loadingEmails.value || emails.value.length >= totalEmails.value) return
    emailsOffset.value += emailsLimit
    await fetchCandidateEmails(true)
  }

  const selectSender = (senderEmail: string | null) => {
    selectedSender.value = senderEmail
    emailsOffset.value = 0
    fetchCandidateEmails(false)
  }

  // Selection actions
  const toggleEmail = (email: EmailMetadata) => {
    const next = new Set(selectedEmailIds.value)
    if (next.has(email.id)) {
      next.delete(email.id)
      selectedEmailsMap.value.delete(email.id)
    } else {
      next.add(email.id)
      selectedEmailsMap.value.set(email.id, email)
    }
    selectedEmailIds.value = next
  }

  const selectAllVisible = () => {
    const next = new Set(selectedEmailIds.value)
    for (const em of emails.value) {
      next.add(em.id)
      selectedEmailsMap.value.set(em.id, em)
    }
    selectedEmailIds.value = next
  }

  const deselectAllVisible = () => {
    const next = new Set(selectedEmailIds.value)
    for (const em of emails.value) {
      next.delete(em.id)
      selectedEmailsMap.value.delete(em.id)
    }
    selectedEmailIds.value = next
  }

  const toggleSelectAllVisible = () => {
    if (isAllVisibleSelected.value) {
      deselectAllVisible()
    } else {
      selectAllVisible()
    }
  }

  const selectAllFromSender = (senderEmail: string) => {
    const next = new Set(selectedEmailIds.value)
    for (const em of emails.value) {
      if (em.from.toLowerCase() === senderEmail.toLowerCase()) {
        next.add(em.id)
        selectedEmailsMap.value.set(em.id, em)
      }
    }
    selectedEmailIds.value = next
  }

  const deselectAll = () => {
    selectedEmailIds.value = new Set()
    selectedEmailsMap.value.clear()
  }

  // CSV export
  const exportSelectedCSV = () => {
    const itemsToExport: EmailMetadata[] = []
    if (selectedEmailIds.value.size > 0) {
      for (const em of selectedEmailsMap.value.values()) {
        if (selectedEmailIds.value.has(em.id)) {
          itemsToExport.push(em)
        }
      }
    } else {
      itemsToExport.push(...emails.value)
    }

    if (itemsToExport.length === 0) return

    const headers = ['Message ID', 'From Address', 'From Name', 'Subject', 'Date', 'Size (Bytes)']
    const rows = itemsToExport.map((em) => [
      `"${(em.id || '').replace(/"/g, '""')}"`,
      `"${(em.from || '').replace(/"/g, '""')}"`,
      `"${(em.fromName || '').replace(/"/g, '""')}"`,
      `"${(em.subject || '').replace(/"/g, '""')}"`,
      `"${em.date || ''}"`,
      String(em.sizeEstimate || 0),
    ])

    const csvContent =
      'data:text/csv;charset=utf-8,' +
      [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
    const encodedUri = encodeURI(csvContent)
    const link = document.createElement('a')
    link.setAttribute('href', encodedUri)
    link.setAttribute(
      'download',
      `deletion_candidates_${new Date().toISOString().slice(0, 10)}.csv`
    )
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // Copy selected IDs
  const copySelectedIDs = async (): Promise<boolean> => {
    if (selectedEmailIds.value.size === 0) return false
    const ids = Array.from(selectedEmailIds.value).join('\n')
    try {
      await navigator.clipboard.writeText(ids)
      return true
    } catch {
      return false
    }
  }

  // Relative age helper
  const getRelativeAge = (dateStr: string): string => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    if (diffDays < 30) {
      return `${diffDays} days ago`
    }
    const diffMonths = Math.floor(diffDays / 30)
    if (diffMonths < 12) {
      return `${diffMonths} ${diffMonths === 1 ? 'month' : 'months'} old`
    }
    const diffYears = (diffDays / 365).toFixed(1)
    return `${diffYears} years old`
  }

  // Watchers to auto-refresh on filter changes
  watch(ageFilter, () => {
    emailsOffset.value = 0
    fetchCandidateEmails(false)
  })

  let searchTimer: number | undefined
  watch(emailSearch, () => {
    clearTimeout(searchTimer)
    searchTimer = window.setTimeout(() => {
      emailsOffset.value = 0
      fetchCandidateEmails(false)
    }, 300)
  })

  return {
    senders,
    filteredSenders,
    loadingSenders,
    sendersError,
    senderSearch,
    emails,
    totalEmails,
    loadingEmails,
    emailsError,
    selectedSender,
    ageFilter,
    emailSearch,
    selectedEmailIds,
    selectedCount,
    selectedSize,
    totalCandidateEmailsCount,
    totalCandidateSizeBytes,
    isAllVisibleSelected,
    fetchCandidateSenders,
    fetchCandidateEmails,
    loadMoreEmails,
    selectSender,
    toggleEmail,
    selectAllVisible,
    deselectAllVisible,
    toggleSelectAllVisible,
    selectAllFromSender,
    deselectAll,
    exportSelectedCSV,
    copySelectedIDs,
    getRelativeAge,
  }
}
