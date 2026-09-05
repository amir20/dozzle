<template>
  <a
    v-if="container.url"
    :href="container.url"
    target="_blank"
    rel="noopener noreferrer"
    class="text-base-content/40 hover:text-primary inline-flex shrink-0 items-center gap-1 transition-colors"
    :title="label"
    :aria-label="label"
    @click.stop
  >
    <slot></slot>
    <mdi:open-in-new class="size-4" />
  </a>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();

const { t } = useI18n();

const label = computed(() => t("tooltip.open-container-url", { url: container.url }));
</script>
