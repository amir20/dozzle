<template>
  <!-- Written so nobody needs the docs: when, which, what each container will do, and
       what keeps it safe. -->
  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("settings.updates") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("auto-update.page-desc") }}</p>
    </div>

    <InlineNotice v-if="status && !status.enableActions" type="info">
      {{ $t("auto-update.needs-actions") }}
      <template #actions>
        <router-link to="/settings/setup" class="btn btn-sm">{{ $t("settings.setup") }}</router-link>
      </template>
    </InlineNotice>

    <!-- Dozzle's own image: what the schedule will do to it, and Update now. -->
    <SelfUpdateStatus v-if="status && scheduled" :status="status" :auto-update="scheduled" />

    <AutoUpdateForm v-if="status" :status="status" autosave />
  </section>

  <section class="flex flex-col gap-3">
    <div class="flex items-end justify-between gap-3">
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
        {{ $t("auto-update.containers") }}
      </h2>
      <button
        v-if="config.imageCheckMode !== 'off'"
        type="button"
        class="btn btn-ghost btn-xs text-base-content/60"
        :disabled="checking"
        @click="checkAll(true)"
      >
        <span v-if="checking" class="loading loading-spinner size-3"></span>
        {{ $t("toolbar.check-for-updates") }}
      </button>
    </div>
    <ContainerUpdatePolicyList />
  </section>

  <section class="flex flex-col gap-3">
    <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
      {{ $t("auto-update.safety-title") }}
    </h2>
    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <div v-for="line in safety" :key="line.key" class="flex items-start gap-3 p-4">
        <div class="bg-info/10 text-info shrink-0 rounded-full p-1.5">
          <component :is="line.icon" class="size-4" />
        </div>
        <p class="text-sm">{{ $t(`auto-update.safety-${line.key}`) }}</p>
      </div>
      <div class="p-2">
        <a
          href="https://dozzle.dev/guide/actions#auto-updating-containers"
          target="_blank"
          rel="noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
        >
          <mdi:book-open-variant class="size-4 opacity-60" />
          <span class="flex-1">{{ $t("auto-update.learn-more") }}</span>
          <mdi:open-in-new class="size-3.5 opacity-40" />
        </a>
      </div>
    </div>
  </section>
</template>

<script lang="ts" setup>
import IconVerify from "~icons/mdi/shield-check-outline";
import IconCleanup from "~icons/mdi/broom";
import IconSkip from "~icons/mdi/pin-outline";

// The layout fetches the status.
const { status } = useSetup();
const { checkAll, checking } = useImageUpdates();

const scheduled = computed(() => {
  const update = status.value?.autoUpdate;
  return update && update.mode !== "off" ? update : undefined;
});

const safety = [
  { key: "verify", icon: IconVerify },
  { key: "cleanup", icon: IconCleanup },
  { key: "skip", icon: IconSkip },
];
</script>
