<template>
  <div>
    <h1>{{ title }}</h1>

    <div class="controls">
      <span class="label">Sort:</span>
      <Link
        v-for="option in sortOptions"
        :key="option.value"
        :href="`/users?sort=${option.value}`"
        class="chip"
        :class="{ active: sort === option.value }"
      >
        {{ option.label }}
      </Link>
    </div>

    <ul class="user-list">
      <li v-for="user in users" :key="user.id" class="user-item">
        <div class="user-name">{{ user.name }}</div>
        <div class="user-meta">#{{ user.id }} · {{ user.role }}</div>
      </li>
    </ul>

    <div class="pager">
      <Link
        v-if="prevPage"
        :href="`/users?sort=${sort}&page=${prevPage}`"
        class="btn secondary"
      >
        Prev
      </Link>
      <Link
        v-if="nextPage"
        :href="`/users?sort=${sort}&page=${nextPage}`"
        class="btn"
        preserve-scroll
        :only="['users', 'page', 'prevPage', 'nextPage', 'totalPages']"
      >
        Load more
      </Link>
    </div>

    <p class="muted">Page {{ page }} of {{ totalPages }}</p>
  </div>
</template>

<script setup>
import { Link } from '@inertiajs/vue3'

defineProps({
  title: String,
  users: Array,
  sort: String,
  page: Number,
  totalPages: Number,
  prevPage: [Number, null],
  nextPage: [Number, null],
})

const sortOptions = [
  { value: 'name', label: 'Name ↑' },
  { value: 'name_desc', label: 'Name ↓' },
  { value: 'id_desc', label: 'ID ↓' },
  { value: 'role', label: 'Role' },
]
</script>

<style scoped>
.controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.label {
  font-weight: 600;
  color: #374151;
}

.chip {
  padding: 0.35rem 0.75rem;
  border-radius: 999px;
  border: 1px solid #e5e7eb;
  text-decoration: none;
  color: #374151;
  font-size: 0.9rem;
  transition: all 0.2s ease;
}

.chip.active {
  background: #eef2ff;
  border-color: #c7d2fe;
  color: #3730a3;
}

.user-list {
  list-style: none;
  padding: 0;
  margin: 0 0 1rem;
}

.user-item {
  padding: 0.75rem 0;
  border-bottom: 1px solid #e5e7eb;
}

.user-name {
  font-weight: 600;
}

.user-meta {
  color: #6b7280;
  font-size: 0.9rem;
}

.pager {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  margin-top: 0.75rem;
}

.btn.secondary {
  background: #e5e7eb;
  color: #111827;
}

.btn.secondary:hover {
  background: #d1d5db;
}

.muted {
  color: #6b7280;
  margin-top: 0.75rem;
}
</style>
