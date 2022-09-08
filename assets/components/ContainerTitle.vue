<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column is-ellipsis">
      {{ container.name }} {{ t("test") }}
      <span class="tag is-dark">{{ container.image.replace(/@sha.*/, "") }}</span>
    </span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/types/Container";
import { inject, ComputedRef } from "vue";
import { useI18n } from "vue-i18n";

const container = inject("container") as ComputedRef<Container>;
const { t } = useI18n();
</script>

<style lang="scss" scoped></style>
