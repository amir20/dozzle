<template>
  <!-- The sidebar tree, full screen. It used to hang off a hamburger at the top of the
       page; now it is one of the tabs, so it sits under the bar rather than over it. -->
  <transition name="browse">
    <div
      v-if="browsing"
      class="bg-base-100 pt-safe fixed inset-0 z-30 flex flex-col"
      :style="{ paddingBottom: 'var(--mobile-tabbar-height)' }"
      data-testid="navigation"
    >
      <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
        <div class="flex shrink-0 items-center gap-2">
          <Logo class="size-9 shrink-0" aria-hidden="true" />
          <span class="truncate text-2xl font-light tracking-tight">Dozzle</span>
          <ProBadge />
        </div>
        <div class="bg-base-content/10 h-px shrink-0"></div>
        <SideMenu class="min-h-0 flex-1" />
      </div>
    </div>
  </transition>

  <nav
    class="fixed inset-x-3 bottom-[calc(0.75rem+env(safe-area-inset-bottom))] z-40 flex items-end gap-2"
    :aria-label="$t('mobile-nav.label')"
  >
    <div class="frosted border-base-content/10 flex flex-1 rounded-full border p-1 shadow-lg">
      <router-link
        :to="{ name: '/' }"
        class="tab-item"
        :class="{ active: active === 'home' }"
        @click="browsing = false"
      >
        <mdi:home-variant-outline class="size-6" />
        <span>{{ $t("mobile-nav.home") }}</span>
      </router-link>
      <button
        type="button"
        class="tab-item"
        :class="{ active: active === 'browse' }"
        :aria-expanded="browsing"
        data-testid="browse"
        @click="browsing = !browsing"
      >
        <mdi:format-list-bulleted class="size-6" />
        <span>{{ $t("mobile-nav.browse") }}</span>
      </button>
      <router-link
        v-if="config.enableNotifications"
        :to="{ name: '/notifications' }"
        class="tab-item relative"
        :class="{ active: active === 'notifications' }"
        @click="browsing = false"
      >
        <mdi:bell-outline class="size-6" />
        <span>{{ $t("title.notifications") }}</span>
        <span
          v-if="unseenAlerts"
          class="bg-warning absolute top-1.5 left-[calc(50%+0.5rem)] size-2 rounded-full"
        ></span>
      </router-link>
      <router-link
        :to="{ name: '/settings' }"
        class="tab-item"
        :class="{ active: active === 'settings' }"
        @click="browsing = false"
      >
        <mdi:cog-outline class="size-6" />
        <span>{{ $t("title.settings") }}</span>
      </router-link>
    </div>

    <!-- Search is its own button beside the bar, the way iOS apps split it off. -->
    <button
      type="button"
      class="frosted border-base-content/10 grid size-15 shrink-0 place-items-center rounded-full border shadow-lg"
      :aria-label="$t('placeholder.search')"
      @click="$emit('search')"
    >
      <mdi:magnify class="size-6" />
    </button>
  </nav>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";

defineEmits<{ search: [] }>();

const route = useRoute();
const browsing = ref(false);
const { unseen: unseenAlerts } = useRecentAlerts();

watch(
  () => route.fullPath,
  () => (browsing.value = false),
);

const active = computed(() => {
  if (browsing.value) return "browse";
  switch (route.name) {
    case "/":
      return "home";
    case "/notifications":
      return "notifications";
    case "/settings":
      return "settings";
    default:
      return undefined;
  }
});
</script>

<style scoped>
@reference "@/main.css";

.tab-item {
  @apply text-base-content/60 flex min-h-13 min-w-0 flex-1 flex-col items-center justify-center gap-0.5 rounded-full px-1 text-[0.625rem] font-medium transition-colors;
}

.tab-item span {
  @apply max-w-full truncate;
}

.tab-item.active {
  @apply bg-base-content/10 text-primary;
}

.browse-enter-active,
.browse-leave-active {
  @apply transition-[opacity,transform] duration-200;
}

.browse-enter-from,
.browse-leave-to {
  @apply translate-y-4 opacity-0;
}

@media (prefers-reduced-motion: reduce) {
  .browse-enter-active,
  .browse-leave-active {
    @apply transition-none;
  }
}
</style>
