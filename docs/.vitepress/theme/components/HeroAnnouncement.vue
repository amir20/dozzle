<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vitepress";

// The one release note worth interrupting the hero for. Copy lives here rather
// than in index.md so the pill does not have to be repeated, and re-translated,
// in five home pages.
const COPY = {
  en: { badge: "v11", label: "A new look, and sign in with GitHub", href: "/guide/whats-new" },
  de: { badge: "v11", label: "Neues Design und Anmeldung mit GitHub", href: "/de/guide/whats-new" },
  fr: { badge: "v11", label: "Nouveau design et connexion avec GitHub", href: "/fr/guide/whats-new" },
  es: { badge: "v11", label: "Nuevo diseño e inicio de sesión con GitHub", href: "/es/guide/whats-new" },
  zh: { badge: "v11", label: "全新界面，支持 GitHub 登录", href: "/zh/guide/whats-new" },
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
