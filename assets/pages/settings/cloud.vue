<template>
  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 mb-1 text-xs font-semibold tracking-wide uppercase">{{ $t("cloud.title") }}</h2>
      <p class="text-base-content/60 text-xs">{{ $t("settings.cloud-desc") }}</p>
    </div>
    <CloudSettingsCard />
  </section>
</template>
