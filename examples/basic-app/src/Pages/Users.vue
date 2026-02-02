<template>
  <div>
    <h1>{{ title }}</h1>
    <p>Users page content.</p>
  </div>
</template>

<script setup>
defineProps({
  title: String,
})
</script>
