import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ContactStatsChart from '../ContactStatsChart.vue'

describe('ContactStatsChart', () => {
  const sampleContacts = [
    {
      email: 'alice@example.com',
      name: 'Alice Cooper',
      receivedCount: 42,
      totalSize: 500000,
    },
    {
      email: 'bob@example.com',
      name: 'Bob Marley',
      receivedCount: 88,
      totalSize: 1200000,
    },
  ]

  it('renders chart title and bars when contacts are provided', () => {
    const wrapper = mount(ContactStatsChart, {
      props: { contacts: sampleContacts, loading: false },
    })

    expect(wrapper.text()).toContain('Top Senders by Emails Received')
    expect(wrapper.find('.chart-container').exists()).toBe(true)
    // Verify TanStack Chart SVG is rendered inside the container
    const svg = wrapper.find('svg.ts-chart')
    expect(svg.exists()).toBe(true)
    // Check that marks/elements are rendered for the contacts
    expect(svg.findAll('path, rect').length).toBeGreaterThan(0)
  })

  it('handles duplicate contact names without collision', () => {
    const duplicateNameContacts = [
      {
        email: 'davemichels@gmail.com',
        name: 'Dave Michels',
        receivedCount: 50,
        totalSize: 1000,
      },
      {
        email: 'dave@michels.com',
        name: 'Dave Michels',
        receivedCount: 30,
        totalSize: 1000,
      },
    ]

    const wrapper = mount(ContactStatsChart, {
      props: { contacts: duplicateNameContacts, loading: false },
    })

    const svg = wrapper.find('svg.ts-chart')
    expect(svg.exists()).toBe(true)
    // Both contacts should be rendered into distinct categories on the axis
    expect(svg.text()).toContain('davemichels')
    expect(svg.text()).toContain('dave')
  })
  it('displays loading message when loading prop is true', () => {
    const wrapper = mount(ContactStatsChart, {
      props: { contacts: [], loading: true },
    })

    expect(wrapper.text()).toContain('Loading chart data...')
  })

  it('displays empty message when contacts array is empty', () => {
    const wrapper = mount(ContactStatsChart, {
      props: { contacts: [], loading: false },
    })

    expect(wrapper.text()).toContain('No sender data available to plot.')
  })
})
