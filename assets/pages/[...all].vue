<template>
  <PageWithLinks>
    <div class="hero bg-base-200 min-h-screen">
      <div class="hero-content text-center">
        <div class="max-w-md">
          <p class="py-6 text-2xl font-bold">{{ $t("error.page-not-found") }}</p>
        </div>
      </div>
    </div>
  </PageWithLinks>
</template>

<script lang="ts" setup>
const { t } = useI18n();
setTitle(t("title.page-not-found"));
</script>
