<template>
  <div class="app-container">
    <h1>Gmail DeepClean</h1>

    <div v-if="!isAuthenticated" class="auth-container">
      <p>To manage your emails, you need to authorize this application.</p>
      <button @click="startAuth" class="btn btn-primary">Connect to Gmail</button>
    </div>

    <div v-else class="email-manager">
      <div class="toolbar">
        <button @click="fetchEmails" class="btn btn-primary">Refresh Emails</button>
        <button @click="logout" class="btn btn-secondary">Logout</button>
      </div>

      <p v-if="loading">Loading emails...</p>
      <p v-else-if="error" class="error">{{ error }}</p>
      <p v-else-if="emails.length === 0">No emails found.</p>
      <email-list v-else :emails="emails" @delete-email="handleDeleteEmail" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import EmailList from './components/EmailList.vue'
import type { Email, OAuthToken } from './types'

export default defineComponent({
  name: 'App',
  components: {
    EmailList,
  },
  setup() {
    const isAuthenticated = ref(false)
    const token = ref<OAuthToken | null>(null)
    const emails = ref<Email[]>([])
    const loading = ref(false)
    const error = ref<string | null>(null)

    const receiveMessage = (event: MessageEvent) => {
      console.log('Received message from popup:', event.origin)

      try {
        // Log the data to help with debugging
        console.log('Message data:', JSON.stringify(event.data))

        if (event.data && event.data.token) {
          // The token is already a parsed object, not a string
          token.value = event.data.token
          isAuthenticated.value = true

          // Store in localStorage
          localStorage.setItem('gmail_token', JSON.stringify(token.value))
          console.log('Authentication successful, fetching emails...')
          fetchEmails()
        } else {
          console.warn('Received message but no token found in data')
        }
      } catch (e) {
        console.error('Error processing auth message:', e)
        error.value = 'Authentication failed. Please try again.'
      }
    }

    onMounted(() => {
      // Check if we have a token in localStorage
      const savedToken = localStorage.getItem('gmail_token')
      if (savedToken) {
        try {
          token.value = JSON.parse(savedToken)
          isAuthenticated.value = true
          fetchEmails()
        } catch (e) {
          console.error('Invalid token in localStorage:', e)
          localStorage.removeItem('gmail_token')
        }
      }

      // Listen for messages from the OAuth popup
      window.addEventListener('message', receiveMessage, false)
    })

    onBeforeUnmount(() => {
      // Clean up event listener
      window.removeEventListener('message', receiveMessage)
    })

    const startAuth = () => {
      // Open OAuth popup
      window.open('/auth/gmail', 'gmail_auth', 'width=600,height=600')
    }

    const fetchEmails = async () => {
      if (!token.value) return

      loading.value = true
      error.value = null

      try {
        const response = await fetch('/api/emails', {
          headers: {
            Authorization: JSON.stringify(token.value),
          },
        })

        if (!response.ok) {
          throw new Error(`HTTP error ${response.status}`)
        }

        const data = await response.json()
        emails.value = data.messages || []
      } catch (err) {
        console.error('Error fetching emails:', err)
        error.value = 'Failed to load emails. Please try again.'

        // If we got a 401, we need to re-authenticate
        if (err instanceof Error && err.message.includes('401')) {
          isAuthenticated.value = false
          localStorage.removeItem('gmail_token')
        }
      } finally {
        loading.value = false
      }
    }

    const handleDeleteEmail = async (id: string) => {
      if (!token.value) return

      try {
        const response = await fetch(`/api/emails/${id}`, {
          method: 'DELETE',
          headers: {
            Authorization: JSON.stringify(token.value),
          },
        })

        if (!response.ok) {
          throw new Error(`HTTP error ${response.status}`)
        }

        // Remove the email from the list
        emails.value = emails.value.filter((email) => email.id !== id)
      } catch (err) {
        console.error('Error deleting email:', err)
        alert('Failed to delete email. Please try again.')
      }
    }

    const logout = () => {
      isAuthenticated.value = false
      token.value = null
      emails.value = []
      localStorage.removeItem('gmail_token')
    }

    return {
      isAuthenticated,
      emails,
      loading,
      error,
      startAuth,
      fetchEmails,
      handleDeleteEmail,
      logout,
    }
  },
})
</script>

<style>
.app-container {
  max-width: 960px;
  margin: 0 auto;
  padding: 20px;
  font-family: Arial, sans-serif;
}

h1 {
  color: #4285f4;
  text-align: center;
}

.auth-container {
  text-align: center;
  margin-top: 100px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 20px;
}

.btn {
  border: none;
  padding: 10px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 16px;
}

.btn-primary {
  background-color: #4285f4;
  color: white;
}

.btn-primary:hover {
  background-color: #3367d6;
}

.btn-secondary {
  background-color: #f1f1f1;
  color: #333;
}

.btn-secondary:hover {
  background-color: #e4e4e4;
}

.error {
  color: red;
  font-weight: bold;
}
</style>
