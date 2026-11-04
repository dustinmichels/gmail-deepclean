import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SenderTable from '../SenderTable.vue'

describe('SenderTable', () => {
  const sampleContacts = [
    {
      email: 'newsletter@promo.com',
      name: 'Promo Deals',
      receivedCount: 15,
      totalSize: 450000,
      lastReceived: '2026-09-28T10:00:00Z',
    },
    {
      email: 'stranger@coldreach.io',
      name: 'Alex Sales',
      receivedCount: 8,
      totalSize: 120000,
      lastReceived: '2026-09-27T15:00:00Z',
    },
  ]

  const defaultProps = {
    contacts: sampleContacts,
    total: 2,
    loading: false,
    searchQuery: '',
    sortField: 'received_count',
  }

  it('renders the never-emailed senders table', () => {
    const wrapper = mount(SenderTable, { props: defaultProps })

    expect(wrapper.text()).toContain('newsletter@promo.com')
    expect(wrapper.text()).toContain('Promo Deals')
    expect(wrapper.text()).toContain('stranger@coldreach.io')
    expect(wrapper.findAll('.sender-table tbody tr')).toHaveLength(2)
  })

  it('emits update:sort when the sort option changes', async () => {
    const wrapper = mount(SenderTable, { props: defaultProps })
    await wrapper.find('.select-sort').setValue('total_size')

    expect(wrapper.emitted('update:sort')![0]).toEqual(['total_size'])
  })

  it('emits select-sender when Emails button is clicked', async () => {
    const wrapper = mount(SenderTable, { props: defaultProps })
    const viewBtn = wrapper.find('.view-btn')
    await viewBtn.trigger('click')

    expect(wrapper.emitted('select-sender')).toBeTruthy()
    expect(wrapper.emitted('select-sender')![0]).toEqual(['newsletter@promo.com'])
  })

  it('emits export-csv when export button clicked', async () => {
    const wrapper = mount(SenderTable, { props: defaultProps })
    const exportBtn = wrapper.find('.btn-export')
    await exportBtn.trigger('click')

    expect(wrapper.emitted('export-csv')).toBeTruthy()
  })

  it('renders error state when error prop is passed and emits retry', async () => {
    const wrapper = mount(SenderTable, {
      props: {
        ...defaultProps,
        contacts: [],
        error: 'Failed to fetch contacts from server',
      },
    })

    expect(wrapper.text()).toContain('Failed to fetch contacts from server')
    expect(wrapper.text()).not.toContain('No senders found.')
    const retryBtn = wrapper.find('.btn-retry')
    expect(retryBtn.exists()).toBe(true)
    await retryBtn.trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
  })

  it('renders genuine empty state when contacts list is empty and error is null', () => {
    const wrapper = mount(SenderTable, {
      props: {
        ...defaultProps,
        contacts: [],
        error: null,
      },
    })

    expect(wrapper.text()).toContain('No senders found.')
    expect(wrapper.find('.error-state').exists()).toBe(false)
  })
})
