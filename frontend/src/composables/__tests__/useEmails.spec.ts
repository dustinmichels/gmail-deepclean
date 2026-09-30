import { describe, expect, it, vi } from 'vitest'
import { useEmails } from '../useEmails'

describe('useEmails composable', () => {
  const authHeaders = () => ({ Authorization: 'Bearer test-token' })

  it('resets emailsFilter to "all" when filterBySender is called', async () => {
    let capturedUrl = ''
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      capturedUrl = url
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [], total: 0 }),
      })
    })

    const { emailsFilter, selectedSender, filterBySender } = useEmails(authHeaders)

    // Simulate user having one-way senders filter active
    emailsFilter.value = 'never_emailed'
    expect(emailsFilter.value).toBe('never_emailed')

    // Click on a sender
    filterBySender('notification+pvl16c6f@facebookmail.com')

    expect(selectedSender.value).toBe('notification+pvl16c6f@facebookmail.com')
    expect(emailsFilter.value).toBe('all')

    // Verify fetch request does not retain filter=never_emailed
    expect(capturedUrl).toContain('sender=notification%2Bpvl16c6f%40facebookmail.com')
    expect(capturedUrl).not.toContain('filter=never_emailed')
  })

  it('captures emailsError when fetchEmails returns non-ok response', async () => {
    globalThis.fetch = vi.fn().mockImplementation(() => {
      return Promise.resolve({
        ok: false,
        status: 500,
        text: () => Promise.resolve('Internal database scan error'),
      })
    })

    const { emailsError, fetchEmails } = useEmails(authHeaders)

    await fetchEmails(false)

    expect(emailsError.value).toBe('Internal database scan error')
  })
})
