<template>
  <div>
    <h1>{{ title }}</h1>
    <p>Settings page content.</p>

    <div class="panel">
      <h2>Optional prop</h2>
      <p class="muted">Diagnostics are loaded only when requested.</p>

      <div v-if="diagnostics" class="diagnostics">
        <pre>{{ diagnostics }}</pre>
      </div>

      <Link
        v-else
        href="/settings"
        class="btn"
        preserve-scroll
        :only="['diagnostics']"
      >
        Load diagnostics
      </Link>
    </div>

    <div class="panel">
      <h2>Precognition form</h2>
      <p class="muted">Validation-only request using Precognition headers.</p>

      <form class="form-grid" @submit.prevent="validateForm">
        <label class="field">
          Name
          <input v-model="form.name" type="text" class="input" />
          <span v-if="errors.name" class="error">{{ errors.name[0] }}</span>
        </label>

        <label class="field">
          Email
          <input v-model="form.email" type="email" class="input" />
          <span v-if="errors.email" class="error">{{ errors.email[0] }}</span>
        </label>

        <div class="actions">
          <button type="submit" class="btn">Validate only</button>
          <span v-if="status" class="status">{{ status }}</span>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import axios from 'axios'
import { Link } from '@inertiajs/vue3'
import { reactive, ref } from 'vue'

const form = reactive({
  name: '',
  email: '',
})

const errors = ref({})
const status = ref('')

const validateForm = async () => {
  status.value = 'Validating...'
  errors.value = {}

  const payload = new URLSearchParams()
  payload.set('name', form.name)
  payload.set('email', form.email)

  try {
    await axios.post('/users/create', payload, {
      headers: {
        Precognition: 'true',
        'Precognition-Validate-Only': 'name,email',
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })
    status.value = 'Looks good'
  } catch (error) {
    if (error.response?.status === 422) {
      errors.value = error.response.data?.errors ?? {}
      status.value = 'Fix the errors above'
      return
    }
    status.value = 'Request failed'
  }
}

defineProps({
  title: String,
  diagnostics: Object,
})
</script>

<style scoped>
.panel {
  margin-top: 1.5rem;
  padding: 1rem 1.25rem;
  border: 1px dashed #e5e7eb;
  border-radius: 10px;
  background: #fff;
}

.diagnostics {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 0.75rem;
  margin: 0.75rem 0;
}

.diagnostics pre {
  margin: 0;
  font-size: 0.85rem;
}

.muted {
  color: #6b7280;
}

.form-grid {
  display: grid;
  gap: 1rem;
  margin-top: 1rem;
}

.field {
  display: grid;
  gap: 0.35rem;
  color: #374151;
  font-weight: 600;
}

.input {
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: #fff;
  font-size: 0.95rem;
}

.input:focus {
  outline: none;
  border-color: #c7d2fe;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
}

.error {
  color: #dc2626;
  font-size: 0.85rem;
  font-weight: 500;
}

.actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.status {
  color: #6b7280;
  font-size: 0.9rem;
}
</style>
