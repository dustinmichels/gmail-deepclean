import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SenderTable from '../SenderTable.vue'

describe('SenderTable', () => {
  const sampleContacts = [
    {
      email: 'newsletter@promo.com',
      name: 'Promo Deals',
      receivedCount: 15,
      sentCount: 0,
      totalSize: 450000,
      lastReceived: '2026-09-28T10:00:00Z',
      neverEmailed: true,
    },
    {
      email: 'colleague@work.com',
      name: 'Bob Colleague',
      receivedCount: 8,
      sentCount: 6,
      totalSize: 120000,
      lastReceived: '2026-09-27T15:00:00Z',
      neverEmailed: false,
    },
  ]

  const defaultProps = {
    contacts: sampleContacts,
    total: 2,
    loading: false,
    activeFilter: 'all',
    searchQuery: '',
    sortField: 'received_count',
    sortOrder: 'desc',
    neverEmailedCount: 1,
  }

  it('renders contacts table and shows warning badge for never-emailed contact', () => {
    const wrapper = mount(SenderTable, { props: defaultProps })

    expect(wrapper.text()).toContain('newsletter@promo.com')
    expect(wrapper.text()).toContain('Promo Deals')
    expect(wrapper.text()).toContain('Never') // Warning badge for 0 sent
    expect(wrapper.text()).toContain('colleague@work.com')
    expect(wrapper.find('.row-warning').exists()).toBe(true)
  })

  it('emits update:filter when filter tabs are clicked', async () => {
    const wrapper = mount(SenderTable, { props: defaultProps })
    const neverTab = wrapper.find('.never-emailed-tab')
    await neverTab.trigger('click')

    expect(wrapper.emitted('update:filter')).toBeTruthy()
    expect(wrapper.emitted('update:filter')![0]).toEqual(['never_emailed'])
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
    expect(wrapper.text()).not.toContain('No contacts found matching the selected filter.')
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

    expect(wrapper.text()).toContain('No contacts found matching the selected filter.')
    expect(wrapper.find('.error-state').exists()).toBe(false)
  })
})
