<template>
  <PageWithLinks>
    <NotFound :title="$t('error.page-not-found')">
      <template #icon><mdi:map-marker-question-outline class="size-5" /></template>
    </NotFound>
  </PageWithLinks>
</template>

<script lang="ts" setup>
const { t } = useI18n();
setTitle(t("title.page-not-found"));
</script>
