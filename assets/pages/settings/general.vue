<template>
  <!-- APPEARANCE: app-wide only. Anything that changes how a log line looks is in Logs,
       under the preview it changes. -->
  <SettingsSection :title="$t('settings.appearance')">
    <div
      class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y overflow-hidden rounded-lg border"
    >
      <SettingRow :label="$t('settings.color-scheme')" class="px-4">
        <div class="flex gap-3">
          <button
            v-for="opt in themes"
            :key="opt.value"
            type="button"
            class="flex flex-col items-center gap-1.5"
            :aria-pressed="lightTheme === opt.value"
            @click="lightTheme = opt.value"
          >
            <span
              class="border-base-content/15 flex h-12 w-20 overflow-hidden rounded-md border transition-shadow"
              :class="{ 'ring-primary ring-offset-base-100 ring-2 ring-offset-2': lightTheme === opt.value }"
            >
              <span
                v-for="swatch in opt.swatches"
                :key="swatch"
                :data-theme="swatch"
                class="bg-base-100 flex flex-1 flex-col gap-1 p-2"
              >
                <span class="bg-base-content/20 h-1 w-3/4 rounded-full"></span>
                <span class="bg-primary h-1 w-1/2 rounded-full"></span>
                <span class="bg-base-content/20 h-1 w-full rounded-full"></span>
              </span>
            </span>
            <span
              class="text-xs"
              :class="lightTheme === opt.value ? 'font-medium' : 'text-base-content/60 font-normal'"
            >
              {{ opt.label }}
            </span>
          </button>
        </div>
      </SettingRow>
      <SettingRow :label="$t('settings.locale')" class="px-4">
        <DropdownMenu
          plain
          v-model="locale"
          :options="[
            { label: 'Auto', value: '' },
            ...availableLocales.map((l) => ({ label: localeName(l), value: l })),
          ]"
        />
      </SettingRow>
    </div>
  </SettingsSection>

  <!-- BEHAVIOR -->
  <SettingsSection :title="$t('settings.behavior')">
    <div
      class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y overflow-hidden rounded-lg border"
    >
      <SettingRow
        :label="$t('settings.automatic-redirect')"
        :description="$t('settings.automatic-redirect-desc')"
        class="px-4"
      >
        <DropdownMenu
          plain
          v-model="automaticRedirect"
          :options="[
            { label: $t('settings.redirect.instant'), value: 'instant' },
            { label: $t('settings.redirect.delayed'), value: 'delayed' },
            { label: $t('settings.redirect.none'), value: 'none' },
          ]"
        />
      </SettingRow>
      <SettingRow tag="label" :label="$t('settings.search')" :description="$t('settings.search-desc')" class="px-4">
        <template #label-suffix><key-shortcut char="f" /></template>
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="search" />
      </SettingRow>
      <SettingRow
        v-if="config.imageCheckMode !== 'off'"
        tag="label"
        :label="$t('settings.show-image-update-alert')"
        :description="$t('settings.show-image-update-alert-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showImageUpdateAlert" />
      </SettingRow>
    </div>
  </SettingsSection>

  <!-- ADVANCED: the two settings almost nobody changes, folded away. -->
  <details
    class="collapse-arrow border-base-content/15 bg-base-200/40 divide-base-content/10 group/advanced collapse divide-y rounded-lg border"
  >
    <summary class="collapse-title text-base-content/70 flex items-center gap-2 text-sm font-medium">
      <span class="flex-1">{{ $t("settings.advanced") }}</span>
      <span class="text-base-content/40 hidden text-xs font-normal group-open/advanced:hidden @xl:inline">
        {{ $t("settings.show-std") }}, {{ $t("settings.small-scrollbars") }}
      </span>
    </summary>
    <div class="collapse-content divide-base-content/10 divide-y p-0">
      <SettingRow tag="label" :label="$t('settings.show-std')" :description="$t('settings.show-std-desc')" class="px-4">
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showStd" />
      </SettingRow>
      <SettingRow
        tag="label"
        :label="$t('settings.small-scrollbars')"
        :description="$t('settings.small-scrollbars-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="smallerScrollbars" />
      </SettingRow>
    </div>
  </details>
</template>

<script lang="ts" setup>
import {
  automaticRedirect,
  lightTheme,
  locale,
  search,
  showImageUpdateAlert,
  showStd,
  smallerScrollbars,
} from "@/stores/settings";
import { availableLocales } from "@/modules/i18n";

// Each language named in itself, e.g. "Deutsch", "日本語".
function localeName(l: string) {
  const name = new Intl.DisplayNames([l], { type: "language" }).of(l) ?? l;
  return name.charAt(0).toLocaleUpperCase(l) + name.slice(1);
}

const { t } = useI18n();

const themes = computed(() => [
  { label: t("settings.theme.light"), value: "light" as const, swatches: ["light"] },
  { label: t("settings.theme.dark"), value: "dark" as const, swatches: ["dark"] },
  { label: t("settings.theme.auto"), value: "auto" as const, swatches: ["light", "dark"] },
]);
</script>
