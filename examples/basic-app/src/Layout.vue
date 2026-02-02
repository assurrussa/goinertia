<template>
  <div>
    <nav>
      <Link 
        v-for="item in menu" 
        :key="item.href" 
        :href="item.href" 
        style="margin-right: 12px"
      >
        {{ item.label }}
      </Link>
    </nav>
    <div v-if="flash.success" style="background: #d4edda; color: #155724; padding: 10px; margin-bottom: 20px;">
      {{ flash.success }}
    </div>
    <div v-if="flash.error" style="background: #f8d7da; color: #721c24; padding: 10px; margin-bottom: 20px;">
      {{ flash.error }}
    </div>
    <main>
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
