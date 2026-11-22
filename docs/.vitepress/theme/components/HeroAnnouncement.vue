<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vitepress";

// The release notes worth interrupting the hero for, newest product news first.
// Copy lives here rather than in index.md so the pills do not have to be
// repeated, and re-translated, in five home pages.
const IOS_URL = "https://cloud.dozzle.dev/ios?from=dozzle_docs";
const COPY = {
  en: [{ badge: "New", label: "Dozzle Cloud for iPhone, private beta", href: IOS_URL }],
  de: [{ badge: "Neu", label: "Dozzle Cloud fürs iPhone, private Beta", href: IOS_URL }],
  fr: [{ badge: "Nouveau", label: "Dozzle Cloud pour iPhone, bêta privée", href: IOS_URL }],
  es: [{ badge: "Nuevo", label: "Dozzle Cloud para iPhone, beta privada", href: IOS_URL }],
  zh: [{ badge: "新", label: "Dozzle Cloud iPhone 版，私人测试", href: IOS_URL }],
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
