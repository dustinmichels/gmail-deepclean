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
      from: 'me@gmail.com',
      fromName: 'Me',
      to: ['client@corp.com'],
      subject: 'Invoice #1042',
      snippet: 'Attached is the invoice for services rendered.',
      date: '2026-09-27T09:15:00Z',
      labelIds: ['SENT'],
      sizeEstimate: 45000,
      isSent: true,
    },
  ]

  const defaultProps = {
    emails: sampleEmails,
    total: 2,
    loading: false,
    selectedSender: null,
    searchQuery: '',
  }

  it('renders email list with subjects, snippets, senders, and direction badges', () => {
    const wrapper = mount(EmailList, { props: defaultProps })

    expect(wrapper.text()).toContain('Quarterly Review Meeting')
    expect(wrapper.text()).toContain('Jane Doe')
    expect(wrapper.text()).toContain('Invoice #1042')
    expect(wrapper.text()).toContain('⬇ Received')
    expect(wrapper.text()).toContain('⬆ Sent')
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
})
