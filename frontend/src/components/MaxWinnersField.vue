<script setup lang="ts">
import { ref } from 'vue'

/** 每題得獎名額。0 代表不限：投稿幾張就能排幾名 */
const model = defineModel<number>({ required: true })

// 切到「全部」再切回來時，回到原本設的名額而不是重設成 5
const lastLimit = ref(model.value > 0 ? model.value : 5)

const toggleAll = () => {
  if (model.value === 0) {
    model.value = lastLimit.value
  } else {
    lastLimit.value = model.value
    model.value = 0
  }
}
</script>

<template>
  <div class="space-y-3">
    <button
      class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
      @click="toggleAll"
    >
      <span>
        <span class="text-sm font-semibold">全部投稿都可以排名</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
          打開後每張照片都能被點選排名；前 5 名 100 / 80 / 60 / 40 / 20 分，第 6 名起 10 分
        </span>
      </span>
      <span
        class="relative h-6 w-11 shrink-0 rounded-full transition"
        :class="model === 0 ? 'bg-blush-500' : 'bg-white/15'"
      >
        <span
          class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
          :class="model === 0 ? 'left-[22px]' : 'left-0.5'"
        />
      </span>
    </button>

    <label v-if="model > 0" class="block">
      <span class="text-sm text-slate-300">每題得獎名額：前 {{ model }} 名</span>
      <input
        :value="model"
        type="range"
        min="1"
        max="20"
        class="mt-2 w-full accent-blush-500"
        @input="model = lastLimit = Number(($event.target as HTMLInputElement).value)"
      />
    </label>
  </div>
</template>
