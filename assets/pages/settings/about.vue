<template>
  <!-- What this install is and whether it is stale. What it will do about that is the
       Updates page. -->
  <section class="flex flex-wrap items-center justify-between gap-3">
    <div class="flex flex-wrap items-center gap-2.5">
      <span class="text-[0.9375rem] font-semibold">Dozzle</span>
      <span class="badge badge-soft badge-sm">{{ config.version }}</span>
      <a
        v-if="hasRelease"
        :href="latestRelease?.htmlUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="badge badge-soft badge-warning badge-sm hover:bg-warning/15"
      >
        <span class="status status-warning"></span>
        {{ $t("settings.new-version", { version: latestRelease?.name }) }}
      </a>
    </div>
    <div class="flex gap-2">
      <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer" class="btn btn-sm">
        <mdi:github /> GitHub
      </a>
      <a href="https://github.com/sponsors/amir20" target="_blank" rel="noopener noreferrer" class="btn btn-sm">
        <mdi:heart class="text-error" /> {{ $t("settings.sponsor") }}
      </a>
    </div>
  </section>

  <!-- The wizard is a guided pass over the server pages, so it can always be run
       again. The setup API only exists in server mode. -->
  <section v-if="server" class="flex flex-col gap-3">
    <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("setup.title") }}</h2>
    <div class="border-base-content/15 bg-base-200/40 flex flex-wrap items-center gap-3 rounded-lg border p-4">
      <span class="bg-base-content/10 text-base-content/70 shrink-0 rounded-full p-2">
        <mdi:rocket-launch-outline class="size-5" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="text-base-content/60 block text-sm">{{ $t("settings.run-setup-desc") }}</span>
      </span>
      <button type="button" class="btn btn-sm shrink-0" @click="openWizard">
        {{ $t("settings.run-setup") }}
      </button>
    </div>
  </section>
</template>

<script lang="ts" setup>
const { latestRelease, hasRelease } = useAnnouncements();
const { openWizard } = useSetup();
const server = config.mode === "server";
</script>
