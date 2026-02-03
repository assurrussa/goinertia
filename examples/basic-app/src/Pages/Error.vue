<template>
  <div class="error-page">
    <div class="error-code">{{ status }}</div>
    <p class="error-message">{{ message || 'Something went wrong' }}</p>

    <Link href="/" class="btn">
      Go Home
    </Link>
  </div>
</template>

<script setup>
import { Link } from '@inertiajs/vue3'

defineProps({
  status: Number,
  message: String,
})
</script>

<style scoped>
.error-page {
  text-align: center;
  padding: 1.5rem 1rem;
}

.error-code {
  font-size: clamp(2.5rem, 6vw, 4rem);
  color: #ef4444;
  margin-bottom: 0.75rem;
  font-weight: 800;
  line-height: 1;
}

.error-message {
  font-size: 1.125rem;
  margin-bottom: 1.75rem;
}
</style>
