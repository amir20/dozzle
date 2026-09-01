<template>
  <div v-if="visible" role="status" class="alert alert-warning m-2 flex items-center gap-2 py-2 text-sm">
    <carbon:upgrade class="size-4 shrink-0" />
    <span class="grow">
      {{ $t("toolbar.update-available") }}
      <span class="opacity-70">{{ container.image }}</span>
    </span>

    <button v-if="updatable" class="btn btn-xs" @click="update()" :disabled="actionStates.update">
      {{ container.isSwarm ? $t("toolbar.update-service") : $t("toolbar.update") }}
    </button>
    <a v-else-if="isSelf" class="btn btn-xs" :href="releaseNotesUrl" target="_blank" rel="noreferrer noopener">
      {{ $t("toolbar.view-release-notes") }}
    </a>

    <button class="btn btn-ghost btn-xs" @click="dismiss()" :title="$t('toolbar.dismiss-update')">
      <mdi:close class="size-4" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();

const containerRef = toRef(() => container);
const { showAlert, updatable, isSelf, dismiss } = useImageUpdate(containerRef);
const { actionStates, update } = useContainerActions(containerRef);

const { latestRelease } = useAnnouncements();
const releaseNotesUrl = computed(() => latestRelease.value?.htmlUrl ?? "https://github.com/amir20/dozzle/releases");

// The banner is opt-in. The dot on the toolbar is the default signal so the
// log view keeps its vertical space.
const visible = computed(() => showAlert.value && showImageUpdateAlert.value);
</script>
