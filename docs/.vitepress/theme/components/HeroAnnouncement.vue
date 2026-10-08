<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vitepress";

// The one release note worth interrupting the hero for. Copy lives here rather
// than in index.md so the pill does not have to be repeated, and re-translated,
// in five home pages.
const COPY = {
  en: { badge: "v11.3", label: "Safe container updates, on a schedule", href: "/guide/updates" },
  de: { badge: "v11.3", label: "Sichere Container-Updates, nach Zeitplan", href: "/de/guide/updates" },
  fr: { badge: "v11.3", label: "Mises à jour sûres des conteneurs, planifiées", href: "/fr/guide/updates" },
  es: { badge: "v11.3", label: "Actualizaciones seguras de contenedores, programadas", href: "/es/guide/updates" },
  zh: { badge: "v11.3", label: "安全的容器更新，支持定时执行", href: "/zh/guide/updates" },
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
    class="mb-5 inline-flex items-center gap-2 rounded-full border border-(--vp-c-divider) bg-(--vp-c-bg-soft) py-1 pr-3 pl-1.5 text-sm font-medium text-(--vp-c-text-2) no-underline! transition-colors hover:border-(--vp-c-brand-1) hover:text-(--vp-c-text-1)"
    :href="copy.href"
  >
    <span class="rounded-full bg-(--vp-c-brand-soft) px-2 py-0.5 text-xs font-semibold text-(--vp-c-brand-1)">
      {{ copy.badge }}
    </span>
    {{ copy.label }}
    <Icon icon="mdi:arrow-right" class="size-4 opacity-60" />
  </a>
</template>
