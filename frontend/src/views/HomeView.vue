<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-8 px-6 py-12">
    <header class="text-center">
      <div class="text-6xl">💄</div>
      <h1 class="mt-4 text-4xl font-black tracking-tight">今日我最美</h1>
      <p class="mt-3 text-sm leading-relaxed text-slate-400">
        出題、拍照、比美。<br />
        主畫面開在電視或筆電，大家用手機掃碼就能玩。
      </p>
    </header>

    <div class="flex flex-col gap-3">
      <button class="btn-primary w-full py-4 text-lg" @click="router.push('/create')">
        🎬 開一間房（主畫面）
      </button>
      <button class="btn-ghost w-full py-4 text-lg" @click="router.push('/join')">
        📱 加入遊戲
      </button>
    </div>

    <section class="card text-sm text-slate-300">
      <h2 class="text-base font-bold text-white">怎麼玩</h2>
      <ol class="mt-3 space-y-2 leading-relaxed">
        <li>1. 開房後主畫面會顯示 QR code，大家掃碼入場。</li>
        <li>2. 每一題出現在手機最上方，馬上拍照上傳。</li>
        <li>3. 照片會即時跳到主畫面，最快的前 5 張留在下方。</li>
        <li>4. 主持人選 1~5 張得獎，然後進下一題。</li>
      </ol>
    </section>

    <p class="text-center text-xs leading-relaxed text-slate-500">
      照片只存在伺服器記憶體，房間結束就整批刪除，不會留檔。
    </p>

    <!-- 後台入口刻意不放連結，要直接打 /admin 才進得去 —— 玩家看不到就不會好奇去點 -->
  </main>
</template>
