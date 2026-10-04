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
</template>

<script lang="ts" setup>
const { latestRelease, hasRelease } = useAnnouncements();
</script>
