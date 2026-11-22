<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vitepress";

// The release notes worth interrupting the hero for, newest product news first.
// Copy lives here rather than in index.md so the pills do not have to be
// repeated, and re-translated, in five home pages.
const IOS_URL = "https://cloud.dozzle.dev/ios?from=dozzle_docs";
const COPY = {
  en: [
    { badge: "New", label: "Dozzle Cloud for iPhone, private beta", href: IOS_URL },
    { badge: "v11.3", label: "Safe container updates, on a schedule", href: "/guide/updates" },
  ],
  de: [
    { badge: "Neu", label: "Dozzle Cloud fürs iPhone, private Beta", href: IOS_URL },
    { badge: "v11.3", label: "Sichere Container-Updates, nach Zeitplan", href: "/de/guide/updates" },
  ],
  fr: [
    { badge: "Nouveau", label: "Dozzle Cloud pour iPhone, bêta privée", href: IOS_URL },
    { badge: "v11.3", label: "Mises à jour sûres des conteneurs, planifiées", href: "/fr/guide/updates" },
  ],
  es: [
    { badge: "Nuevo", label: "Dozzle Cloud para iPhone, beta privada", href: IOS_URL },
    { badge: "v11.3", label: "Actualizaciones seguras de contenedores, programadas", href: "/es/guide/updates" },
  ],
  zh: [
    { badge: "新", label: "Dozzle Cloud iPhone 版，私人测试", href: IOS_URL },
    { badge: "v11.3", label: "安全的容器更新，支持定时执行", href: "/zh/guide/updates" },
  ],
} as const;

type Locale = keyof typeof COPY;

const route = useRoute();
const items = computed(() => {
  const segment = route.path.split("/")[1] as Locale;
  return COPY[segment] ?? COPY.en;
});
</script>

<template>
  <div class="mb-5 flex flex-wrap gap-2">
    <a
      v-for="item in items"
      :key="item.href"
      class="inline-flex max-w-full items-center gap-2 rounded-full border border-(--vp-c-divider) bg-(--vp-c-bg-soft) py-1 pr-3 pl-1.5 text-left text-xs font-medium text-(--vp-c-text-2) no-underline! transition-colors hover:border-(--vp-c-brand-1) hover:text-(--vp-c-text-1) sm:text-sm"
      :href="item.href"
    >
      <span class="rounded-full bg-(--vp-c-brand-soft) px-2 py-0.5 text-xs font-semibold text-(--vp-c-brand-1)">
        {{ item.badge }}
      </span>
      <span class="min-w-0">{{ item.label }}</span>
      <Icon icon="mdi:arrow-right" class="size-4 shrink-0 opacity-60" />
    </a>
  </div>
</template>
