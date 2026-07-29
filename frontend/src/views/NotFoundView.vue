<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <main class="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
    <div class="text-6xl">🔍</div>
    <h1 class="text-2xl font-bold">找不到這個頁面</h1>
    <button class="btn-primary" @click="router.push('/')">回首頁</button>
  </main>
</template>
