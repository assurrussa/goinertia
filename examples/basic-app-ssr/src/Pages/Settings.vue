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
  </div>
</template>

<script setup>
import { Link } from '@inertiajs/vue3'

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
</style>
