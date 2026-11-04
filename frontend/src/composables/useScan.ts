import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { OverallStats, ScanProgress } from '../types'

export function useScan(authHeaders: () => Record<string, string>) {
  const progress = ref<ScanProgress>({
    userId: '',
    userEmail: '',
    isProcessing: false,
    totalEmails: 0,
    estimatedTotal: 0,
    statusMessage: 'Ready to scan',
  })

  const stats = ref<OverallStats>({
    totalEmailsProcessed: 0,
    totalReceived: 0,
    totalSent: 0,
    uniqueSenders: 0,
    totalSizeBytes: 0,
    neverEmailedSenders: 0,
    neverEmailedEmails: 0,
    neverEmailedSizeBytes: 0,
    localDbSizeBytes: 0,
  })

  const isScanning = computed(() => progress.value.isProcessing)
  let pollInterval: number | null = null

  const fetchStatus = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    try {
      const res = await fetch('/api/inbox/status', { headers })
      if (res.ok) {
        const data: ScanProgress = await res.json()
        progress.value = data
      }
    } catch (e) {
      console.error('Failed to fetch scan status:', e)
    }
  }

  const fetchStats = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    try {
      const res = await fetch('/api/inbox/stats', { headers })
      if (res.ok) {
        stats.value = await res.json()
      }
    } catch (e) {
      console.error('Failed to fetch stats:', e)
    }
  }

  const startPolling = () => {
    clearInterval(pollInterval ?? undefined)
    pollInterval = window.setInterval(async () => {
      await fetchStatus()
      await fetchStats()
      if (!progress.value.isProcessing) {
        stopPolling()
      }
    }, 1000)
  }
  const stopPolling = () => {
    clearInterval(pollInterval ?? undefined)
    pollInterval = null
  }

  const startScan = async (reset = false) => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    // Optimistically indicate scan has started for immediate UI response
    progress.value = {
      ...progress.value,
      isProcessing: true,
      statusMessage: reset ? 'Starting new inbox scan...' : 'Resuming scan...',
    }
    startPolling()

    try {
      const res = await fetch(`/api/inbox/process?reset=${reset}`, {
        method: 'POST',
        headers,
      })
      if (res.ok) {
        progress.value = await res.json()
        await fetchStats()
      } else {
        await fetchStatus()
      }
    } catch (e) {
      console.error('Failed to start scan:', e)
      await fetchStatus()
    }
  }

  const pauseScan = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    try {
      const res = await fetch('/api/inbox/pause', {
        method: 'POST',
        headers,
      })
      if (res.ok) {
        progress.value = await res.json()
        stopPolling()
        await fetchStats()
      }
    } catch (e) {
      console.error('Failed to pause scan:', e)
    }
  }

  watch(
    () => progress.value.isProcessing,
    (processing) => {
      if (processing) {
        startPolling()
      } else {
        stopPolling()
      }
    }
  )

  onBeforeUnmount(() => {
    stopPolling()
  })

  return {
    progress,
    stats,
    isScanning,
    startScan,
    pauseScan,
    fetchStatus,
    fetchStats,
  }
}
