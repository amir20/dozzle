<template>
  <!-- Why server settings are read-only here, the same in every setup step and every
       server page in Settings. Renders nothing when they can be changed. -->
  <InlineNotice v-if="!status.dataPersisted" type="warning">{{ $t("setup.error.no-data") }}</InlineNotice>
  <template v-else-if="!status.canWrite">
    <!-- A form that knows its env vars names them, and stays quiet once every one is set. -->
    <template v-if="envs && status.authProvider === 'none'">
      <InlineNotice v-if="envs.length" type="info">
        <i18n-t keypath="setup.actions.window-closed-env" tag="span">
          <template #envs>
            <template v-for="part in envParts" :key="part.key">
              <code v-if="part.env" class="font-mono">{{ part.value }}</code>
              <template v-else>{{ part.value }}</template>
            </template>
          </template>
        </i18n-t>
      </InlineNotice>
    </template>
    <InlineNotice v-else type="info">{{ $t(setupAccessMessageKey(status)) }}</InlineNotice>
  </template>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

const { envs } = defineProps<{
  status: SetupStatus;
  // The env vars that would change what this form shows, for the no-login notice.
  envs?: string[];
}>();

const { locale } = useI18n();

// "A and B" in the reader's language, with each name kept in mono.
const envParts = computed(() =>
  new Intl.ListFormat(locale.value, { type: "conjunction" })
    .formatToParts(envs ?? [])
    .map((part, i) => ({ key: i, env: part.type === "element", value: part.value })),
);
</script>
