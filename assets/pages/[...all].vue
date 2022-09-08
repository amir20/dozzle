<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container has-text-centered">
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
