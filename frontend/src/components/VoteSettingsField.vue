<script setup lang="ts">
/** 投票模式設定：開關、候選張數、投票倒數秒數。建房與大廳共用 */
const enabled = defineModel<boolean>('enabled', { required: true })
const candidates = defineModel<number>('candidates', { required: true })
const seconds = defineModel<number>('seconds', { required: true })
</script>

<template>
  <div class="space-y-3">
    <button
      class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
      @click="enabled = !enabled"
    >
      <span>
        <span class="text-sm font-semibold">投票模式</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
          玩家用手機投票（不能投自己），票數排名決定得獎、同票時先交的排前面；最快的前 {{ candidates }} 張是候選，
          湊滿就開始 {{ seconds }} 秒投票倒數
        </span>
      </span>
      <span
        class="relative h-6 w-11 shrink-0 rounded-full transition"
        :class="enabled ? 'bg-blush-500' : 'bg-white/15'"
      >
        <span
          class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
          :class="enabled ? 'left-[22px]' : 'left-0.5'"
        />
      </span>
    </button>

    <template v-if="enabled">
      <label class="block">
        <span class="text-sm text-slate-300">候選張數：最快的前 {{ candidates }} 張</span>
        <input
          :value="candidates"
          type="range"
          min="2"
          max="20"
          class="mt-2 w-full accent-blush-500"
          @input="candidates = Number(($event.target as HTMLInputElement).value)"
        />
      </label>
      <label class="block">
        <span class="text-sm text-slate-300">投票倒數：{{ seconds }} 秒</span>
        <input
          :value="seconds"
          type="range"
          min="5"
          max="60"
          class="mt-2 w-full accent-blush-500"
          @input="seconds = Number(($event.target as HTMLInputElement).value)"
        />
      </label>
    </template>
  </div>
</template>
