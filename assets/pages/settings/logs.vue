<template>
  <!-- LOGS -->
  <SettingsSection :title="$t('settings.logs')">
    <div
      class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y overflow-hidden rounded-lg border"
    >
      <div ref="previewEl">
        <LogList
          :messages="fakeMessages"
          :last-selected-item="undefined"
          :show-container-name="false"
          class="bg-base-100 pb-2"
        />
      </div>
      <SettingRow :label="$t('settings.font-size')" class="px-4">
        <span role="tablist" class="tabs tabs-box tabs-sm">
          <button
            v-for="opt in sizes"
            :key="opt.value"
            type="button"
            role="tab"
            class="tab"
            :class="{ 'tab-active': size === opt.value }"
            @click="size = opt.value"
          >
            {{ opt.label }}
          </button>
        </span>
      </SettingRow>
      <SettingRow tag="label" :label="$t('settings.compact')" :description="$t('settings.compact-desc')" class="px-4">
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="compact" />
      </SettingRow>
      <SettingRow
        tag="label"
        :label="$t('settings.show-timestamps')"
        :description="$t('settings.show-timestamps-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showTimestamp" />
      </SettingRow>
      <SettingRow
        tag="label"
        :label="$t('settings.soft-wrap')"
        :description="$t('settings.soft-wrap-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="softWrap" />
      </SettingRow>
      <SettingRow
        tag="label"
        :label="$t('settings.highlight-errors')"
        :description="$t('settings.highlight-errors-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="highlightErrors" />
      </SettingRow>
      <!-- One row each: two unlabelled "Auto" menus side by side read as the same thing. -->
      <SettingRow :label="$t('settings.date-format')" class="px-4">
        <DropdownMenu
          plain
          v-model="dateLocale"
          :options="[
            { label: $t('settings.hour.auto'), value: 'auto' },
            { label: 'MM/DD/YYYY', value: 'en-US' },
            { label: 'DD/MM/YYYY', value: 'en-GB' },
            { label: 'DD.MM.YYYY', value: 'de-DE' },
            { label: 'YYYY-MM-DD', value: 'en-CA' },
          ]"
        />
      </SettingRow>
      <SettingRow :label="$t('settings.time-format')" class="px-4">
        <DropdownMenu
          plain
          v-model="hourStyle"
          :options="[
            { label: $t('settings.hour.auto'), value: 'auto' },
            { label: $t('settings.hour.12'), value: '12' },
            { label: $t('settings.hour.24'), value: '24' },
          ]"
        />
      </SettingRow>
    </div>
  </SettingsSection>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry, GroupedLogEntry } from "@/models/LogEntry";
import { compact, dateLocale, hourStyle, showTimestamp, size, softWrap, highlightErrors } from "@/stores/settings";
import { i18n } from "@/modules/i18n";

const { t } = useI18n();

const sizes = computed(() => [
  { label: t("settings.size.small"), value: "small" as const },
  { label: t("settings.size.medium"), value: "medium" as const },
  { label: t("settings.size.large"), value: "large" as const },
]);

// The preview sits above the controls that resize it, so every change would push
// the control out from under the pointer. Scroll by however much the preview's
// bottom edge moved, which keeps everything below it still.
const previewEl = useTemplateRef("previewEl");
let previewBottom: number | undefined;
const previewSettings = () => [size.value, compact.value, showTimestamp.value, softWrap.value];
watch(previewSettings, () => (previewBottom = previewEl.value?.getBoundingClientRect().bottom), { flush: "pre" });
watch(
  previewSettings,
  () => {
    if (previewBottom === undefined || !previewEl.value) return;
    window.scrollBy(0, previewEl.value.getBoundingClientRect().bottom - previewBottom);
    previewBottom = undefined;
  },
  { flush: "post" },
);

const now = new Date();
const hoursAgo = (hours: number) => {
  const date = new Date(now);
  date.setHours(date.getHours() - hours);
  return date;
};

const fakeMessages = computedWithControl(
  () => i18n.global.locale.value,
  () => [
    new SimpleLogEntry(t("settings.log.preview"), "123", 1, hoursAgo(16), "info", "stdout", ""),
    new SimpleLogEntry(t("settings.log.warning"), "123", 2, hoursAgo(12), "warn", "stdout", ""),
    new GroupedLogEntry(
      [
        t("settings.log.multi-line-error.start-line"),
        t("settings.log.multi-line-error.middle-line"),
        t("settings.log.multi-line-error.end-line"),
      ],
      "123",
      3,
      hoursAgo(7),
      "error",
      "stderr",
    ),
    new ComplexLogEntry(
      {
        message: t("settings.log.complex"),
        context: {
          key: "value",
          key2: "value2",
        },
      },
      "123",
      6,
      new Date(),
      "info",
      "stdout",
      "",
    ),
    new SimpleLogEntry(t("settings.log.simple"), "123", 7, new Date(), "debug", "stderr", ""),
  ],
);
</script>
