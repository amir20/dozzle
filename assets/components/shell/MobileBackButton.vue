<template>
  <button
    type="button"
    class="icon-btn btn btn-ghost btn-circle shrink-0"
    :aria-label="$t('mobile-nav.back')"
    :title="$t('mobile-nav.back')"
    @click="goBack(router)"
  >
    <mdi:chevron-left class="size-7" />
  </button>
</template>

<script lang="ts" setup>
const router = useRouter();
</script>
