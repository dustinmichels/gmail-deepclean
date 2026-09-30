import { ref, watch } from 'vue'
import type { ContactStat, ContactsResponse, EmailMetadata, EmailsResponse } from '../types'

export function useEmails(authHeaders: () => Record<string, string>) {
  // Contacts State
  const contacts = ref<ContactStat[]>([])
  const totalContacts = ref(0)
  const contactsFilter = ref<string>('all')
  const contactsSearch = ref<string>('')
  const contactsSort = ref<string>('received_count')
  const contactsLoading = ref(false)

  // Emails State
  const emails = ref<EmailMetadata[]>([])
  const totalEmails = ref(0)
  const selectedSender = ref<string | null>(null)
  const emailsFilter = ref<'all' | 'never_emailed'>('all')
  const emailsSearch = ref<string>('')
  const emailsLoading = ref(false)
  const emailsOffset = ref(0)
  const emailsLimit = 25

  const fetchContacts = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    contactsLoading.value = true
    try {
      const params = new URLSearchParams({
        filter: contactsFilter.value,
        search: contactsSearch.value,
        sort: contactsSort.value,
        order: 'desc',
        limit: '100',
        offset: '0',
      })

      const res = await fetch(`/api/inbox/contacts?${params.toString()}`, { headers })
      if (res.ok) {
        const data: ContactsResponse = await res.json()
        contacts.value = data.contacts || []
        totalContacts.value = data.total || 0
      }
    } catch (e) {
      console.error('Failed to fetch contacts:', e)
    } finally {
      contactsLoading.value = false
    }
  }

  const fetchEmails = async (append = false) => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    emailsLoading.value = true
    try {
      if (!append) {
        emailsOffset.value = 0
      }

      const params = new URLSearchParams({
        limit: String(emailsLimit),
        offset: String(emailsOffset.value),
      })

      if (selectedSender.value) {
        params.set('sender', selectedSender.value)
      }
      if (emailsFilter.value && emailsFilter.value !== 'all') {
        params.set('filter', emailsFilter.value)
      }
      if (emailsSearch.value) {
        params.set('search', emailsSearch.value)
      }
      const res = await fetch(`/api/emails?${params.toString()}`, { headers })
      if (res.ok) {
        const data: EmailsResponse = await res.json()
        if (append) {
          emails.value = [...emails.value, ...(data.emails || [])]
        } else {
          emails.value = data.emails || []
        }
        totalEmails.value = data.total || 0
      }
    } catch (e) {
      console.error('Failed to fetch emails:', e)
    } finally {
      emailsLoading.value = false
    }
  }

  const loadMoreEmails = async () => {
    if (emailsLoading.value || emails.value.length >= totalEmails.value) return
    emailsOffset.value += emailsLimit
    await fetchEmails(true)
  }

  const filterBySender = (senderEmail: string) => {
    selectedSender.value = senderEmail
    emailsSearch.value = ''
    fetchEmails(false)
  }

  const clearSenderFilter = () => {
    selectedSender.value = null
    fetchEmails(false)
  }

  const exportContactsCSV = () => {
    if (contacts.value.length === 0) return

    const headers = [
      'Email',
      'Name',
      'Received Count',
      'Sent Count',
      'Total Size (Bytes)',
      'Never Emailed Back',
      'Last Received',
    ]
    const rows = contacts.value.map((c) => [
      `"${c.email.replace(/"/g, '""')}"`,
      `"${(c.name || '').replace(/"/g, '""')}"`,
      c.receivedCount,
      c.sentCount,
      c.totalSize,
      c.neverEmailed ? 'YES' : 'NO',
      c.lastReceived || '',
    ])

    const csvContent = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.setAttribute('href', url)
    link.setAttribute('download', `gmail_contacts_${contactsFilter.value}.csv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  // Watchers to auto-refresh on filter changes
  watch([contactsFilter, contactsSort], () => {
    fetchContacts()
  })

  watch(emailsFilter, () => {
    fetchEmails(false)
  })
  // Debounced search watcher for contacts
  let contactSearchTimer: number | null = null
  watch(contactsSearch, () => {
    clearTimeout(contactSearchTimer ?? undefined)
    contactSearchTimer = window.setTimeout(() => {
      fetchContacts()
    }, 300)
  })

  // Debounced search watcher for emails
  let emailSearchTimer: number | null = null
  watch(emailsSearch, () => {
    clearTimeout(emailSearchTimer ?? undefined)
    emailSearchTimer = window.setTimeout(() => {
      fetchEmails(false)
    }, 300)
  })

  return {
    contacts,
    totalContacts,
    contactsFilter,
    contactsSearch,
    contactsSort,
    contactsLoading,
    emails,
    totalEmails,
    selectedSender,
    emailsFilter,
    emailsSearch,
    emailsLoading,
    fetchContacts,
    fetchEmails,
    loadMoreEmails,
    filterBySender,
    clearSenderFilter,
    exportContactsCSV,
  }
}
