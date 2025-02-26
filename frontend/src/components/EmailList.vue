<template>
  <div class="email-list">
    <div v-for="email in emailsWithDetails" :key="email.id" class="email-item">
      <div class="email-header" @click="toggleEmailDetails(email.id)">
        <div class="email-snippet">
          <!-- In a real app, you'd display more meaningful data from the Gmail API -->
          <span class="email-subject">Email ID: {{ email.id }}</span>
          <span class="email-preview">Click to view details</span>
        </div>
        <button class="delete-btn" @click.stop="confirmDelete(email.id)">Delete</button>
      </div>

      <div v-if="email.showDetails" class="email-details">
        <p>This would display the full email content.</p>
        <p>
          In a production app, you would fetch the full email details from the Gmail API and display
          them here.
        </p>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref } from 'vue'
import type { Email } from '../types'

export default defineComponent({
  name: 'EmailList',
  props: {
    emails: {
      type: Array as () => Email[],
      required: true,
    },
  },
  emits: ['delete-email'],
  setup(props, { emit }) {
    const detailsState = ref<Record<string, boolean>>({})

    const emailsWithDetails = computed(() =>
      props.emails.map((email) => ({
        ...email,
        showDetails: !!detailsState.value[email.id],
      }))
    )

    const toggleEmailDetails = (id: string) => {
      detailsState.value = {
        ...detailsState.value,
        [id]: !detailsState.value[id],
      }
    }

    const confirmDelete = (id: string) => {
      if (confirm('Are you sure you want to delete this email?')) {
        emit('delete-email', id)
      }
    }

    return {
      emailsWithDetails,
      toggleEmailDetails,
      confirmDelete,
    }
  },
})
</script>

<style scoped>
.email-list {
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: hidden;
}

.email-item {
  border-bottom: 1px solid #eee;
}

.email-item:last-child {
  border-bottom: none;
}

.email-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  cursor: pointer;
  background-color: #f9f9f9;
}

.email-header:hover {
  background-color: #f4f4f4;
}

.email-snippet {
  display: flex;
  flex-direction: column;
}

.email-subject {
  font-weight: bold;
  margin-bottom: 5px;
}

.email-preview {
  color: #666;
  font-size: 0.9em;
}

.email-details {
  padding: 15px;
  background-color: #fff;
  border-top: 1px solid #eee;
}

.delete-btn {
  background-color: #db4437;
  color: white;
  border: none;
  padding: 5px 10px;
  border-radius: 4px;
  cursor: pointer;
}

.delete-btn:hover {
  background-color: #c53929;
}
</style>
