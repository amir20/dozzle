<template>
  <!-- Why server settings are read-only here, the same in every setup step and every
       server page in Settings. Renders nothing when they can be changed. -->
  <InlineNotice v-if="!status.dataPersisted" type="warning">{{ $t("setup.error.no-data") }}</InlineNotice>
  <InlineNotice v-else-if="!status.canWrite" type="info">
    {{ $t(setupAccessMessageKey(status)) }}
  </InlineNotice>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

defineProps<{ status: SetupStatus }>();
</script>
