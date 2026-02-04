<template>
  <div>
    <h1>{{ title }}</h1>
    <p>Welcome to the Home page.</p>
    <a href="/undefined-page">Example link to a non-existent page</a>

    <div class="panel">
      <h2>Once prop</h2>
      <p class="muted">Plan: <strong>{{ plan }}</strong></p>
    </div>

    <div class="panel">
      <h2>Deferred prop</h2>
      <Deferred data="heavy">
        <template #fallback>
          <div class="muted">Loading heavy data…</div>
        </template>

        <ul class="list">
          <li v-for="item in heavy" :key="item">{{ item }}</li>
        </ul>
      </Deferred>
    </div>
  </div>
</template>

<script setup>
import { Deferred } from '@inertiajs/vue3'

defineProps({
  title: String,
  plan: String,
  heavy: Array,
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

.list {
  margin: 0.5rem 0 0;
  padding-left: 1.25rem;
}

.muted {
  color: #6b7280;
  margin-top: 0.25rem;
}
</style>
