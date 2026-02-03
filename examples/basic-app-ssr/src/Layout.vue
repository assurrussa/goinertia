<template>
  <div class="app-container">
    <nav>
      <Link 
        v-for="item in menu" 
        :key="item.href" 
        :href="item.href"
        :class="{ 'active': $page.url === item.href }"
      >
        {{ item.label }}
      </Link>
    </nav>
    
    <div v-if="flash.success" class="flash flash-success">
      {{ flash.success }}
    </div>
    
    <div v-if="flash.error" class="flash flash-error">
      {{ flash.error }}
    </div>
    
    <main class="card">
      <slot />
    </main>
  </div>
</template>

<script setup>
import { Link, usePage } from '@inertiajs/vue3'
import { computed } from 'vue'

const page = usePage()
const menu = computed(() => page.props.menu)
const flash = computed(() => page.props.flash || {})
</script>
