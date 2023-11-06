<template>
  <page-with-links>
    <div class="hero min-h-screen bg-base-200">
      <div class="hero-content text-center">
        <div class="max-w-md">
          <p class="py-6 text-2xl font-bold">{{ $t("error.page-not-found") }}</p>
        </div>
      </div>
    </div>
  </page-with-links>
</template>

<script lang="ts" setup>
const { t } = useI18n();
setTitle(t("title.page-not-found"));
</script>
