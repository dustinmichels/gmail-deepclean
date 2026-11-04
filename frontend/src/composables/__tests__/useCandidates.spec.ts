import { describe, expect, it, vi } from 'vitest'
import type { EmailMetadata } from '../../types'
import { useCandidates } from '../useCandidates'

describe('useCandidates composable', () => {
  const authHeaders = () => ({ Authorization: 'Bearer test-token' })

  const mockEmail1: EmailMetadata = {
    id: 'msg-1',
    threadId: 'th-1',
    from: 'old-promo@marketing.com',
    fromName: 'Promo',
    to: ['me@test.com'],
    subject: 'Old Promo',
    snippet: 'Buy now',
    date: '2024-01-01T12:00:00Z',
    sizeEstimate: 2048,
    isSent: false,
    labelIds: ['INBOX'],
  }

  const mockEmail2: EmailMetadata = {
    id: 'msg-2',
    threadId: 'th-2',
    from: 'stranger@coldreach.io',
    fromName: 'Stranger',
    to: ['me@test.com'],
    subject: 'Cold Pitch',
    snippet: 'Quick intro',
    date: '2024-02-01T12:00:00Z',
    sizeEstimate: 4096,
    isSent: false,
    labelIds: ['INBOX'],
  }

  it('fetches candidate senders from /api/inbox/candidate-senders', async () => {
    const mockSenders = [
      {
        email: 'old-promo@marketing.com',
        name: 'Promo',
        candidateCount: 5,
        candidateSize: 10240,
      },
    ]

    let capturedUrl = ''
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      capturedUrl = url
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockSenders),
      })
    })

    const { senders, totalCandidateEmailsCount, totalCandidateSizeBytes, fetchCandidateSenders } =
      useCandidates(authHeaders)

    await fetchCandidateSenders()

    expect(capturedUrl).toBe('/api/inbox/candidate-senders')
    expect(senders.value).toHaveLength(1)
    expect(totalCandidateEmailsCount.value).toBe(5)
    expect(totalCandidateSizeBytes.value).toBe(10240)
  })

  it('fetches candidate emails with filter=candidates by default', async () => {
    let capturedUrl = ''
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      capturedUrl = url
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [mockEmail1, mockEmail2], total: 2 }),
      })
    })

    const { emails, totalEmails, fetchCandidateEmails } = useCandidates(authHeaders)

    await fetchCandidateEmails()

    expect(capturedUrl).toContain('filter=candidates')
    expect(emails.value).toHaveLength(2)
    expect(totalEmails.value).toBe(2)
  })

  it('updates filter parameter when ageFilter changes', async () => {
    let capturedUrl = ''
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      capturedUrl = url
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [mockEmail1], total: 1 }),
      })
    })

    const { ageFilter, fetchCandidateEmails } = useCandidates(authHeaders)

    ageFilter.value = '6m'
    await fetchCandidateEmails()

    expect(capturedUrl).toContain('filter=candidates_6m')
  })

  it('handles email multi-selection and calculates selected count and size', () => {
    const {
      emails,
      selectedEmailIds,
      selectedCount,
      selectedSize,
      toggleEmail,
      selectAllVisible,
      deselectAllVisible,
      isAllVisibleSelected,
      deselectAll,
    } = useCandidates(authHeaders)

    emails.value = [mockEmail1, mockEmail2]

    expect(selectedCount.value).toBe(0)
    expect(selectedSize.value).toBe(0)
    expect(isAllVisibleSelected.value).toBe(false)

    // Toggle single email
    toggleEmail(mockEmail1)
    expect(selectedEmailIds.value.has('msg-1')).toBe(true)
    expect(selectedCount.value).toBe(1)
    expect(selectedSize.value).toBe(2048)

    // Select all visible
    selectAllVisible()
    expect(selectedCount.value).toBe(2)
    expect(selectedSize.value).toBe(6144)
    expect(isAllVisibleSelected.value).toBe(true)

    // Deselect all visible
    deselectAllVisible()
    expect(selectedCount.value).toBe(0)
    expect(selectedSize.value).toBe(0)

    // Select and deselect all
    toggleEmail(mockEmail2)
    expect(selectedCount.value).toBe(1)
    deselectAll()
    expect(selectedCount.value).toBe(0)
  })

  it('selects all emails from a specific sender', () => {
    const { emails, selectedEmailIds, selectedCount, selectAllFromSender } =
      useCandidates(authHeaders)

    emails.value = [mockEmail1, mockEmail2]

    selectAllFromSender('old-promo@marketing.com')

    expect(selectedCount.value).toBe(1)
    expect(selectedEmailIds.value.has('msg-1')).toBe(true)
    expect(selectedEmailIds.value.has('msg-2')).toBe(false)
  })

  it('copies selected IDs to clipboard', async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeTextMock },
      configurable: true,
    })

    const { toggleEmail, copySelectedIDs } = useCandidates(authHeaders)

    // Nothing selected
    const resultEmpty = await copySelectedIDs()
    expect(resultEmpty).toBe(false)
    expect(writeTextMock).not.toHaveBeenCalled()

    // Select one email
    toggleEmail(mockEmail1)
    const result = await copySelectedIDs()
    expect(result).toBe(true)
    expect(writeTextMock).toHaveBeenCalledWith('msg-1')
  })

  it('computes relative age string accurately', () => {
    const { getRelativeAge } = useCandidates(authHeaders)

    expect(getRelativeAge('')).toBe('')
    const twoMonthsAgo = new Date()
    twoMonthsAgo.setDate(twoMonthsAgo.getDate() - 65)
    expect(getRelativeAge(twoMonthsAgo.toISOString())).toContain('months old')

    const twoYearsAgo = new Date()
    twoYearsAgo.setDate(twoYearsAgo.getDate() - 750)
    expect(getRelativeAge(twoYearsAgo.toISOString())).toContain('years old')
  })
})
