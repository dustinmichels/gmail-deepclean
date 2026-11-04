import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ScanDashboard from '../ScanDashboard.vue'

describe('ScanDashboard', () => {
  const defaultProps = {
    progress: {
      userId: 'test@gmail.com',
      userEmail: 'test@gmail.com',
      isProcessing: false,
      totalEmails: 120,
      statusMessage: 'Ready',
      lastScannedAt: '2026-09-29T12:00:00Z',
    },
    stats: {
      totalEmailsProcessed: 120,
      totalReceived: 80,
      totalSent: 40,
      uniqueSenders: 35,
      totalSizeBytes: 2048576,
      neverEmailedSenders: 18,
      neverEmailedEmails: 45,
      neverEmailedSizeBytes: 512144,
    },
    isScanning: false,
  }

  it('renders overall metrics correctly', () => {
    const wrapper = mount(ScanDashboard, { props: defaultProps })

    expect(wrapper.text()).toContain('120')
    expect(wrapper.text()).toContain('35')
    expect(wrapper.text()).toContain('Resume Scan')
  })

  it('shows the share of all mail and space that comes from never-emailed senders', () => {
    const wrapper = mount(ScanDashboard, { props: defaultProps })
    const card = wrapper.find('.never-emailed-card')

    const pcts = card.findAll('.share-pct').map((el) => el.text())
    expect(pcts).toEqual(['37.5%', '25%'])
    expect(card.text()).toContain('45 of 120 emails')
    expect(card.findAll('.share-fill')[0].attributes('style')).toContain('width: 37.5%')
  })

  it('shows 0% shares before anything has been scanned', () => {
    const wrapper = mount(ScanDashboard, {
      props: {
        ...defaultProps,
        stats: {
          ...defaultProps.stats,
          totalEmailsProcessed: 0,
          totalSizeBytes: 0,
          neverEmailedEmails: 0,
          neverEmailedSizeBytes: 0,
        },
      },
    })

    const pcts = wrapper.findAll('.never-emailed-card .share-pct').map((el) => el.text())
    expect(pcts).toEqual(['0%', '0%'])
  })

  it('displays Start Inbox Scan when totalEmailsProcessed is 0', () => {
    const wrapper = mount(ScanDashboard, {
      props: {
        ...defaultProps,
        stats: { ...defaultProps.stats, totalEmailsProcessed: 0 },
      },
    })
    expect(wrapper.text()).toContain('Start Inbox Scan')
  })

  it('emits start-scan when scan button clicked', async () => {
    const wrapper = mount(ScanDashboard, { props: defaultProps })
    const btn = wrapper.find('.btn-primary')
    await btn.trigger('click')

    expect(wrapper.emitted('start-scan')).toBeTruthy()
    expect(wrapper.emitted('start-scan')![0]).toEqual([false])
  })

  it('shows pause button and emits pause-scan when scanning is true', async () => {
    const wrapper = mount(ScanDashboard, {
      props: {
        ...defaultProps,
        isScanning: true,
        progress: { ...defaultProps.progress, isProcessing: true, statusMessage: 'Scanning...' },
      },
    })

    const pauseBtn = wrapper.find('.btn-warning')
    expect(pauseBtn.exists()).toBe(true)
    expect(pauseBtn.text()).toContain('Pause Scan')

    await pauseBtn.trigger('click')
    expect(wrapper.emitted('pause-scan')).toBeTruthy()
  })

  it('renders progress bar and percentage when scanning with estimatedTotal', () => {
    const wrapper = mount(ScanDashboard, {
      props: {
        ...defaultProps,
        isScanning: true,
        progress: {
          ...defaultProps.progress,
          isProcessing: true,
          totalEmails: 50,
          estimatedTotal: 200,
          statusMessage: 'Scanning email 50 of ~200...',
        },
      },
    })

    expect(wrapper.find('.progress-track').exists()).toBe(true)
    expect(wrapper.find('.progress-fill').exists()).toBe(true)
    expect(wrapper.text()).toContain('25%')
    expect(wrapper.text()).toContain('50')
    expect(wrapper.text()).toContain('200')
  })

  it('renders storage metrics distinguishing cloud space from local db', () => {
    const wrapper = mount(ScanDashboard, {
      props: {
        ...defaultProps,
        stats: {
          ...defaultProps.stats,
          totalSizeBytes: 9500000000,
          localDbSizeBytes: 31457280,
        },
      },
    })

    expect(wrapper.text()).toContain('Gmail Space Analyzed')
    expect(wrapper.text()).toContain('Cloud')
    expect(wrapper.text()).toContain('Local DB:')
    expect(wrapper.text()).toContain('30 MB')
    expect(wrapper.text()).toContain('Metadata-only scan')
  })
})
