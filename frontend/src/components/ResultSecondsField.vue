<script setup lang="ts">
/** 公布得獎畫面停幾秒自動下一題，0 = 手動按「下一題」 */
const model = defineModel<number>({ required: true })

const options = [1, 2, 3, 5, 8, 0]
</script>

<template>
  <div>
    <span class="text-sm text-slate-300">得獎畫面停留</span>
    <span class="ml-2 text-xs text-slate-500">
      {{ model > 0 ? `${model} 秒後自動下一題，也可以提早按` : '每題都由你按「下一題」' }}
    </span>
    <div class="mt-2 flex flex-wrap gap-2">
      <button
        v-for="sec in options"
        :key="sec"
        class="rounded-full px-3 py-1.5 text-xs font-medium transition"
        :class="model === sec ? 'bg-blush-500 text-white' : 'bg-white/5 text-slate-400 hover:bg-white/10'"
        @click="model = sec"
      >
        {{ sec > 0 ? `${sec} 秒` : '手動' }}
      </button>
    </div>
  </div>
</template>
