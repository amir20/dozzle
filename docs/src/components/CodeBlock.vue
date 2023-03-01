<script setup lang="ts">
const source = ref('')
const body = $ref<HTMLElement>()
onMounted(() => {
  source.value = body?.textContent?.trim() || ''
})

const { copy, copied, isSupported } = useClipboard({ source })
</script>

<template>
  <div flex mx-1 gap-1 w-full>
    <code ref="body" class="not-prose" overflow="x-auto" whitespace-nowrap font-mono text="sm lg:base" @click="copy()">
      <slot />
    </code>
    <a
      v-if="isSupported"
      icon-btn
      ml-auto
      :class="copied ? 'i-mdi-clipboard-check' : 'i-mdi-clipboard-outline'"
      text-3xl
      @click="copy()"
    />
  </div>
</template>
