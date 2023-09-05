<template>
  <div class="is-halfheight hero">
    <div class="hero-body">
      <div class="has-text-centered container">
        <h1 class="title">
          404.
          <small class="subtitle">{{ $t("error.page-not-found") }}</small>
        </h1>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { t } = useI18n();
setTitle(t("title.page-not-found"));
</script>
