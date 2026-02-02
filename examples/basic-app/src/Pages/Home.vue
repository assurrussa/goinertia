<template>
  <div>
    <h1>{{ title }}</h1>
    <p>Welcome to the Home page.</p>
  </div>
</template>

<script setup>
defineProps({
  title: String,
})
</script>
