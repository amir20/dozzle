<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vitepress";

// Marks what only works with Dozzle Cloud linked. The copy lives here, like
// HeroAnnouncement's, so a translated page only carries <CloudBadge />.
const COPY = {
  en: { title: "Available with Dozzle Cloud", href: "/guide/dozzle-cloud" },
  de: { title: "Verfügbar mit Dozzle Cloud", href: "/de/guide/dozzle-cloud" },
  fr: { title: "Disponible avec Dozzle Cloud", href: "/fr/guide/dozzle-cloud" },
  es: { title: "Disponible con Dozzle Cloud", href: "/es/guide/dozzle-cloud" },
  zh: { title: "Dozzle Cloud 专属功能", href: "/zh/guide/dozzle-cloud" },
} as const;

type Locale = keyof typeof COPY;

const route = useRoute();
const copy = computed(() => {
  const segment = route.path.split("/")[1] as Locale;
  return COPY[segment] ?? COPY.en;
});
</script>

<template>
  <a
    class="ignore-header inline-flex size-5 items-center justify-center rounded-full bg-(--vp-c-brand-soft) align-middle text-(--vp-c-brand-1) no-underline! transition-colors hover:text-(--vp-c-brand-2)"
    :href="copy.href"
    :title="copy.title"
    :aria-label="copy.title"
  >
    <Icon icon="mdi:cloud-outline" class="size-3.5" />
  </a>
</template>
