<template>
  <section class="flex flex-col gap-4">
    <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("settings.setup") }}</h2>
    <SetupSettingsCard :status="status" />
  </section>
</template>

<script lang="ts" setup>
// The layout fetches the status, so the card reports what is already configured
// without anyone opening the wizard.
const { status } = useSetup();
</script>
