import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { SessionInfo } from '../types'

const SESSION_STORAGE_KEY = 'deepclean_session_token'
const USER_EMAIL_KEY = 'deepclean_user_email'

export function useAuth() {
  const sessionToken = ref<string | null>(null)
  const userEmail = ref<string | null>(null)
  const loading = ref(true)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!sessionToken.value)

  const authHeaders = computed<Record<string, string>>(() => {
    const headers: Record<string, string> = {}
    if (sessionToken.value) {
      headers.Authorization = `Bearer ${sessionToken.value}`
    }
    return headers
  })

  const receiveMessage = (event: MessageEvent) => {
    // Accept messages from same origin, Vite dev server, or Go backend server
    const allowedOrigins = [
      window.location.origin,
      'http://localhost:8080',
      'http://127.0.0.1:8080',
      'http://localhost:5173',
      'http://127.0.0.1:5173',
    ]
    if (!allowedOrigins.includes(event.origin)) return

    if (event.data && event.data.type === 'GMAIL_AUTH_SUCCESS' && event.data.session) {
      const session: SessionInfo = event.data.session
      sessionToken.value = session.sessionToken
      userEmail.value = session.userEmail
      localStorage.setItem(SESSION_STORAGE_KEY, session.sessionToken)
      localStorage.setItem(USER_EMAIL_KEY, session.userEmail)
      error.value = null
    }
  }

  const checkSession = async () => {
    const savedToken = localStorage.getItem(SESSION_STORAGE_KEY)
    const savedEmail = localStorage.getItem(USER_EMAIL_KEY)

    if (!savedToken) {
      loading.value = false
      return
    }

    sessionToken.value = savedToken
    userEmail.value = savedEmail

    try {
      const res = await fetch('/api/auth/me', {
        headers: {
          Authorization: `Bearer ${savedToken}`,
        },
      })

      if (res.ok) {
        const data = await res.json()
        userEmail.value = data.userEmail
        localStorage.setItem(USER_EMAIL_KEY, data.userEmail)
      } else {
        // Session expired or invalid
        logout()
      }
    } catch (e) {
      console.error('Session validation error:', e)
    } finally {
      loading.value = false
    }
  }

  const startAuth = () => {
    error.value = null
    window.open('/auth/gmail', 'gmail_auth', 'width=600,height=700')
  }

  const logout = async () => {
    if (sessionToken.value) {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${sessionToken.value}`,
          },
        })
      } catch (e) {
        console.error('Logout error:', e)
      }
    }

    sessionToken.value = null
    userEmail.value = null
    localStorage.removeItem(SESSION_STORAGE_KEY)
    localStorage.removeItem(USER_EMAIL_KEY)
  }

  onMounted(() => {
    checkSession()
    window.addEventListener('message', receiveMessage)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('message', receiveMessage)
  })

  return {
    isAuthenticated,
    userEmail,
    loading,
    error,
    authHeaders,
    startAuth,
    logout,
  }
}
