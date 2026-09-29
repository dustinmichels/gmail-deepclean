import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { OverallStats, ScanProgress } from '../types'

export function useScan(authHeaders: () => Record<string, string>) {
  const progress = ref<ScanProgress>({
    userId: '',
    userEmail: '',
    isProcessing: false,
    totalEmails: 0,
    statusMessage: 'Ready to scan',
  })

  const stats = ref<OverallStats>({
    totalEmailsProcessed: 0,
    totalReceived: 0,
    totalSent: 0,
    uniqueSenders: 0,
    neverEmailedCount: 0,
    totalSizeBytes: 0,
  })

  const isScanning = computed(() => progress.value.isProcessing)
  let pollInterval: number | null = null

  const fetchStatus = async () => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    try {
      const res = await fetch('/api/inbox/status', { headers })
      if (res.ok) {
        progress.value = await res.json()
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

  const startScan = async (reset = false) => {
    const headers = authHeaders()
    if (!headers.Authorization) return

    try {
      const res = await fetch(`/api/inbox/process?reset=${reset}`, {
        method: 'POST',
        headers,
      })
      if (res.ok) {
        progress.value = await res.json()
        startPolling()
      }
    } catch (e) {
      console.error('Failed to start scan:', e)
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

  const startPolling = () => {
    clearInterval(pollInterval ?? undefined)
    pollInterval = setInterval(async () => {
      await fetchStatus()
      await fetchStats()
      if (!progress.value.isProcessing) {
        stopPolling()
      }
    }, 2000)
  }

  const stopPolling = () => {
    if (pollInterval) {
      clearInterval(pollInterval)
      pollInterval = null
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
