import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { CandidateSender, EmailMetadata } from '../../types'
import ManageCandidates from '../ManageCandidates.vue'

describe('ManageCandidates', () => {
  const mockSender: CandidateSender = {
    email: 'old-promo@deals.com',
    name: 'Weekly Deals',
    candidateCount: 4,
    candidateSize: 8192,
    oldestDate: '2024-01-01T12:00:00Z',
    newestDate: '2024-02-01T12:00:00Z',
  }

  const mockEmail: EmailMetadata = {
    id: 'cand-msg-1',
    threadId: 'th-1',
    from: 'old-promo@deals.com',
    fromName: 'Weekly Deals',
    to: ['me@test.com'],
    subject: 'Huge Savings This Weekend',
    snippet: 'Get 50% off all clearance items...',
    date: '2024-01-15T10:00:00Z',
    sizeEstimate: 2048,
    isSent: false,
    labelIds: ['INBOX'],
  }

  const defaultProps = {
    senders: [mockSender],
    filteredSenders: [mockSender],
    loadingSenders: false,
    sendersError: null,
    senderSearch: '',
    emails: [mockEmail],
    totalEmails: 1,
    loadingEmails: false,
    emailsError: null,
    selectedSender: null,
    ageFilter: 'all' as const,
    emailSearch: '',
    selectedEmailIds: new Set<string>(),
    selectedCount: 0,
    selectedSize: 0,
    totalCandidateEmailsCount: 4,
    totalCandidateSizeBytes: 8192,
    isAllVisibleSelected: false,
  }

  it('renders mandatory read-only safety banner', () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    expect(wrapper.text()).toContain('Candidate Evaluation Mode')
    expect(wrapper.text()).toContain('Strictly Read-Only — DO NOT DELETE YET')
    expect(wrapper.text()).toContain('From a sender you have never emailed back')
    expect(wrapper.text()).toContain('Received more than a month ago')
  })

  it('renders candidate KPI metrics', () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    expect(wrapper.text()).toContain('4') // totalCandidateEmailsCount
    expect(wrapper.text()).toContain('1') // senders.length
    expect(wrapper.text()).toContain('8 KB') // totalCandidateSizeBytes
  })

  it('renders candidate senders and emits selectSender on click', async () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    expect(wrapper.text()).toContain('old-promo@deals.com')
    expect(wrapper.text()).toContain('Weekly Deals')

    const senderItem = wrapper.find('.sender-main')
    expect(senderItem.exists()).toBe(true)

    await senderItem.trigger('click')
    expect(wrapper.emitted('selectSender')).toBeTruthy()
    expect(wrapper.emitted('selectSender')![0]).toEqual(['old-promo@deals.com'])
  })

  it('renders candidate email cards with badges and emits toggleEmail on checkbox click', async () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    expect(wrapper.text()).toContain('Huge Savings This Weekend')
    expect(wrapper.text()).toContain('Never Emailed')

    const checkbox = wrapper.find('.checkbox-btn')
    expect(checkbox.exists()).toBe(true)

    await checkbox.trigger('click')
    expect(wrapper.emitted('toggleEmail')).toBeTruthy()
    expect(wrapper.emitted('toggleEmail')![0]).toEqual([mockEmail])
  })

  it('emits toggleSelectAllVisible when batch select button is clicked', async () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    const batchBtn = wrapper.find('.selection-bar button')
    expect(batchBtn.exists()).toBe(true)

    await batchBtn.trigger('click')
    expect(wrapper.emitted('toggleSelectAllVisible')).toBeTruthy()
  })

  it('ensures the delete action button is strictly disabled', () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    const deleteBtn = wrapper.find('.btn-danger-disabled')
    expect(deleteBtn.exists()).toBe(true)
    expect(deleteBtn.attributes('disabled')).toBeDefined()
    expect(deleteBtn.text()).toContain('Delete Selected (Disabled)')
  })

  it('emits update:ageFilter when filter pill buttons are clicked', async () => {
    const wrapper = mount(ManageCandidates, { props: defaultProps })
    const pillBtns = wrapper.findAll('.pill-btn')
    expect(pillBtns.length).toBeGreaterThan(1)

    // Click on >3 Months Old
    await pillBtns[1].trigger('click')
    expect(wrapper.emitted('update:ageFilter')).toBeTruthy()
    expect(wrapper.emitted('update:ageFilter')![0]).toEqual(['3m'])
  })

  it('renders empty state when no emails are present', () => {
    const wrapper = mount(ManageCandidates, {
      props: {
        ...defaultProps,
        emails: [],
        totalEmails: 0,
      },
    })
    expect(wrapper.text()).toContain('No Candidate Emails Found')
  })
})
