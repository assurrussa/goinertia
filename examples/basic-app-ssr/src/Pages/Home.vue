<template>
  <div>
    <h1>{{ title }}</h1>
    <p>Welcome to the Home page.</p>
    <a href="/undefined-page">Example link to a non-existent page</a>
  </div>
</template>

<script setup>
defineProps({
  title: String,
})
</script>
