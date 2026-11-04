import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import EmailList from '../EmailList.vue'

describe('EmailList', () => {
  const sampleEmails = [
    {
      id: 'email-123',
      threadId: 'th-123',
      from: 'sender@example.com',
      fromName: 'Jane Doe',
      to: ['me@gmail.com'],
      subject: 'Quarterly Review Meeting',
      snippet: 'Here is the agenda for our review meeting tomorrow...',
      date: '2026-09-28T14:30:00Z',
      labelIds: ['INBOX', 'IMPORTANT'],
      sizeEstimate: 12500,
      isSent: false,
    },
    {
      id: 'email-456',
      threadId: 'th-456',
      from: 'billing@vendor.com',
      fromName: 'Vendor Billing',
      to: ['me@gmail.com'],
      subject: 'Invoice #1042',
      snippet: 'Attached is the invoice for services rendered.',
      date: '2026-09-27T09:15:00Z',
      labelIds: ['INBOX'],
      sizeEstimate: 45000,
      isSent: false,
    },
  ]

  const defaultProps = {
    emails: sampleEmails,
    total: 2,
    loading: false,
    selectedSender: null,
    searchQuery: '',
  }

  it('renders email list with subjects, snippets, and senders', () => {
    const wrapper = mount(EmailList, { props: defaultProps })

    expect(wrapper.text()).toContain('Quarterly Review Meeting')
    expect(wrapper.text()).toContain('Jane Doe')
    expect(wrapper.text()).toContain('Invoice #1042')
  })

  it('toggles expanded details when an email card is clicked', async () => {
    const wrapper = mount(EmailList, { props: defaultProps })
    const card = wrapper.find('.email-card')

    expect(wrapper.find('.card-expanded-details').exists()).toBe(false)
    await card.trigger('click')
    expect(wrapper.find('.card-expanded-details').exists()).toBe(true)
    expect(wrapper.text()).toContain('email-123')
    expect(wrapper.text()).toContain('th-123')
  })

  it('displays filter chip when sender is selected and emits clear-sender', async () => {
    const wrapper = mount(EmailList, {
      props: {
        ...defaultProps,
        selectedSender: 'sender@example.com',
      },
    })

    expect(wrapper.text()).toContain('Filtered by sender: sender@example.com')
    const clearBtn = wrapper.find('.btn-clear-filter')
    expect(clearBtn.exists()).toBe(true)

    await clearBtn.trigger('click')
    expect(wrapper.emitted('clear-sender')).toBeTruthy()
  })

  it('emits update:search when typing in search input', async () => {
    const wrapper = mount(EmailList, { props: defaultProps })
    const input = wrapper.find('.input-search')
    await input.setValue('Meeting')

    expect(wrapper.emitted('update:search')).toBeTruthy()
    expect(wrapper.emitted('update:search')![0]).toEqual(['Meeting'])
  })

  it('displays how many emails were received from the sender', () => {
    const wrapper = mount(EmailList, {
      props: {
        ...defaultProps,
        emails: [
          {
            id: 'email-special',
            threadId: 'th-spec',
            from: 'unreplied@vendor.com',
            fromName: 'Vendor Deals',
            to: ['me@gmail.com'],
            subject: 'Flash Sale',
            snippet: 'Don’t miss this sale!',
            date: '2026-09-28T14:30:00Z',
            labelIds: ['INBOX'],
            sizeEstimate: 5000,
            isSent: false,
            senderEmailCount: 24,
          },
        ],
        total: 1,
      },
    })

    expect(wrapper.find('.sender-count-badge').text()).toContain('24 received')
  })

  it('renders error state and emits retry when error prop is set', async () => {
    const wrapper = mount(EmailList, {
      props: {
        ...defaultProps,
        error: 'Failed to retrieve emails: internal server error',
      },
    })

    expect(wrapper.find('.error-state').exists()).toBe(true)
    expect(wrapper.text()).toContain('Failed to retrieve emails: internal server error')
    expect(wrapper.find('.empty-state').exists()).toBe(false)

    const retryBtn = wrapper.find('.error-state .btn-retry')
    expect(retryBtn.exists()).toBe(true)
    await retryBtn.trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
  })
})
