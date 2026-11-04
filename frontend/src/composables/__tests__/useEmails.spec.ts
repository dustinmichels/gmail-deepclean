import { describe, expect, it, vi } from 'vitest'
import { useEmails } from '../useEmails'

describe('useEmails composable', () => {
  const authHeaders = () => ({ Authorization: 'Bearer test-token' })

  it('filterBySender fetches that sender with an encoded address and clears search', async () => {
    let capturedUrl = ''
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      capturedUrl = url
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [], total: 0 }),
      })
    })

    const { emailsSearch, selectedSender, filterBySender } = useEmails(authHeaders)
    emailsSearch.value = 'invoice'

    filterBySender('notification+pvl16c6f@facebookmail.com')

    expect(selectedSender.value).toBe('notification+pvl16c6f@facebookmail.com')
    expect(emailsSearch.value).toBe('')
    expect(capturedUrl).toContain('sender=notification%2Bpvl16c6f%40facebookmail.com')
    expect(capturedUrl).not.toContain('search=')
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
