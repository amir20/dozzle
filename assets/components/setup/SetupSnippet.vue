<template>
  <div class="bg-base-300 relative rounded-lg">
    <pre class="overflow-x-auto p-3 pr-11 font-mono text-xs leading-relaxed">{{ code }}</pre>
    <button
      type="button"
      class="btn btn-ghost btn-sm btn-square absolute top-1 right-1"
      :aria-label="$t('setup.copy')"
      :title="$t('setup.copy')"
      @click="copy(code, { manual: code })"
    >
      <mdi:check v-if="copied" class="text-success size-4" />
      <mdi:content-copy v-else class="size-4 opacity-60" />
    </button>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ code: string }>();
const { copy, copied } = useCopy();
</script>
