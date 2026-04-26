<template>
  <div class="settings-page flex flex-col gap-5 px-4 py-4 md:px-8">
    <section>
      <Links>
        <template #more-items>
          <Tag>{{ config.version }}</Tag>
        </template>
      </Links>
    </section>

    <div class="settings-content">
      <!-- ABOUT -->
      <section class="s-section">
        <div class="s-section-h">
          <h2>{{ $t("settings.about") }}</h2>
        </div>
        <p class="s-section-desc">
          <span v-html="$t('settings.using-version', { version: config.version })"></span>
          <template v-if="hasRelease">
            &nbsp;<span
              v-html="
                $t('settings.update-available', { nextVersion: latestRelease?.name, href: latestRelease?.htmlUrl })
              "
            ></span>
          </template>
        </p>

        <div class="s-card">
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.help-support") }}</div>
            </div>
            <div class="s-row-control s-sponsor-row">
              <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer">
                <mdi:github /> amir20/dozzle
              </a>
              <span class="s-sep">·</span>
              <a href="https://github.com/sponsors/amir20" target="_blank" rel="noopener noreferrer">
                Sponsor on GitHub
              </a>
              <span class="s-sep">·</span>
              <a href="https://buymeacoffee.com/amirraminfar" target="_blank" rel="noopener noreferrer">
                <mdi:beer /> Buy me a beer
              </a>
            </div>
          </div>
        </div>
      </section>

      <!-- CLOUD -->
      <section class="s-section">
        <div class="s-section-h">
          <h2>{{ $t("cloud.title") }}</h2>
        </div>
        <div class="s-card s-card-pad">
          <CloudSettingsCard />
        </div>
      </section>

      <!-- DISPLAY -->
      <section class="s-section">
        <div class="s-section-h">
          <h2>{{ $t("settings.display") }}</h2>
        </div>

        <div class="s-display-grid">
          <div class="s-card">
            <div class="s-row">
              <div class="s-row-label">
                <div class="s-row-title">{{ $t("settings.compact") }}</div>
              </div>
              <div class="s-row-control">
                <button
                  class="s-toggle"
                  :class="{ on: compact }"
                  role="switch"
                  :aria-checked="compact"
                  @click="compact = !compact"
                ></button>
              </div>
            </div>
            <div class="s-row">
              <div class="s-row-label">
                <div class="s-row-title">{{ $t("settings.small-scrollbars") }}</div>
              </div>
              <div class="s-row-control">
                <button
                  class="s-toggle"
                  :class="{ on: smallerScrollbars }"
                  role="switch"
                  :aria-checked="smallerScrollbars"
                  @click="smallerScrollbars = !smallerScrollbars"
                ></button>
              </div>
            </div>
            <div class="s-row">
              <div class="s-row-label">
                <div class="s-row-title">{{ $t("settings.show-timestamps") }}</div>
              </div>
              <div class="s-row-control">
                <button
                  class="s-toggle"
                  :class="{ on: showTimestamp }"
                  role="switch"
                  :aria-checked="showTimestamp"
                  @click="showTimestamp = !showTimestamp"
                ></button>
              </div>
            </div>
            <div class="s-row">
              <div class="s-row-label">
                <div class="s-row-title">{{ $t("settings.show-std") }}</div>
              </div>
              <div class="s-row-control">
                <button
                  class="s-toggle"
                  :class="{ on: showStd }"
                  role="switch"
                  :aria-checked="showStd"
                  @click="showStd = !showStd"
                ></button>
              </div>
            </div>
            <div class="s-row">
              <div class="s-row-label">
                <div class="s-row-title">{{ $t("settings.soft-wrap") }}</div>
              </div>
              <div class="s-row-control">
                <button
                  class="s-toggle"
                  :class="{ on: softWrap }"
                  role="switch"
                  :aria-checked="softWrap"
                  @click="softWrap = !softWrap"
                ></button>
              </div>
            </div>
          </div>

          <LogList
            :messages="fakeMessages"
            :last-selected-item="undefined"
            :show-container-name="false"
            class="s-preview"
          />
        </div>
      </section>

      <!-- OPTIONS -->
      <section class="s-section">
        <div class="s-section-h">
          <h2>{{ $t("settings.options") }}</h2>
        </div>

        <div class="s-card">
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.locale") }}</div>
            </div>
            <div class="s-row-control">
              <DropdownMenu
                v-model="locale"
                :options="[
                  { label: 'Auto', value: '' },
                  ...availableLocales.map((l) => ({ label: l.toLocaleUpperCase(), value: l })),
                ]"
              />
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.datetime-format") }}</div>
            </div>
            <div class="s-row-control s-row-control-tight">
              <DropdownMenu
                v-model="dateLocale"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: 'MM/DD/YYYY', value: 'en-US' },
                  { label: 'DD/MM/YYYY', value: 'en-GB' },
                  { label: 'DD.MM.YYYY', value: 'de-DE' },
                  { label: 'YYYY-MM-DD', value: 'en-CA' },
                ]"
              />
              <DropdownMenu
                v-model="hourStyle"
                :options="[
                  { label: $t('settings.hour.auto'), value: 'auto' },
                  { label: $t('settings.hour.12'), value: '12' },
                  { label: $t('settings.hour.24'), value: '24' },
                ]"
              />
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.font-size") }}</div>
            </div>
            <div class="s-row-control">
              <div class="s-seg">
                <button
                  v-for="opt in [
                    { label: $t('settings.size.small'), value: 'small' },
                    { label: $t('settings.size.medium'), value: 'medium' },
                    { label: $t('settings.size.large'), value: 'large' },
                  ]"
                  :key="opt.value"
                  :class="{ active: size === opt.value }"
                  @click="size = opt.value as typeof size"
                >
                  {{ opt.label }}
                </button>
              </div>
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.color-scheme") }}</div>
            </div>
            <div class="s-row-control">
              <div class="s-seg">
                <button
                  v-for="opt in [
                    { label: $t('settings.theme.light'), value: 'light' },
                    { label: $t('settings.theme.dark'), value: 'dark' },
                    { label: $t('settings.theme.auto'), value: 'auto' },
                  ]"
                  :key="opt.value"
                  :class="{ active: lightTheme === opt.value }"
                  @click="lightTheme = opt.value as typeof lightTheme"
                >
                  {{ opt.label }}
                </button>
              </div>
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.automatic-redirect") }}</div>
            </div>
            <div class="s-row-control">
              <DropdownMenu
                v-model="automaticRedirect"
                :options="[
                  { label: $t('settings.redirect.instant'), value: 'instant' },
                  { label: $t('settings.redirect.delayed'), value: 'delayed' },
                  { label: $t('settings.redirect.none'), value: 'none' },
                ]"
              />
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.group-containers") }}</div>
            </div>
            <div class="s-row-control">
              <DropdownMenu
                v-model="groupContainers"
                :options="[
                  { label: $t('settings.grouping.always'), value: 'always' },
                  { label: $t('settings.grouping.at-least-2'), value: 'at-least-2' },
                  { label: $t('settings.grouping.never'), value: 'never' },
                ]"
              />
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">
                {{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut>
              </div>
            </div>
            <div class="s-row-control">
              <button
                class="s-toggle"
                :class="{ on: search }"
                role="switch"
                :aria-checked="search"
                @click="search = !search"
              ></button>
            </div>
          </div>
          <div class="s-row">
            <div class="s-row-label">
              <div class="s-row-title">{{ $t("settings.show-stopped-containers") }}</div>
            </div>
            <div class="s-row-control">
              <button
                class="s-toggle"
                :class="{ on: showAllContainers }"
                role="switch"
                :aria-checked="showAllContainers"
                @click="showAllContainers = !showAllContainers"
              ></button>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry, GroupedLogEntry } from "@/models/LogEntry";

import {
  automaticRedirect,
  compact,
  hourStyle,
  dateLocale,
  lightTheme,
  search,
  showAllContainers,
  showStd,
  showTimestamp,
  size,
  smallerScrollbars,
  softWrap,
  locale,
  groupContainers,
} from "@/stores/settings";

import { availableLocales, i18n } from "@/modules/i18n";

const { t } = useI18n();

setTitle(t("title.settings"));
const { latestRelease, hasRelease } = useAnnouncements();

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

<style scoped>
@reference "@/main.css";

.settings-content {
  width: 100%;
}

.s-section {
  padding: 24px 0 8px;
}
.s-section:first-of-type {
  padding-top: 0;
}

.s-section-h {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin: 0 0 4px;
}
.s-section-h h2 {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.01em;
  margin: 0;
}
.s-section-desc {
  @apply text-base-content/60;
  font-size: 13px;
  margin: 0 0 16px;
}
.s-section-desc :deep(a) {
  @apply text-primary;
}
.s-section-desc :deep(a:hover) {
  text-decoration: underline;
  text-underline-offset: 4px;
}

.s-card {
  @apply border-base-content/15 bg-base-200/40;
  border-width: 1px;
  border-radius: 10px;
  overflow: hidden;
}
.s-card + .s-card {
  margin-top: 12px;
}
.s-card-pad {
  padding: 16px;
}

.s-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 16px;
  align-items: center;
  padding: 13px 16px;
  min-height: 52px;
  @apply border-base-content/10;
  border-bottom-width: 1px;
}
.s-row:last-child {
  border-bottom-width: 0;
}
.s-row-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.s-row-title {
  font-size: 13.5px;
  font-weight: 500;
}
.s-row-control {
  display: flex;
  align-items: center;
  gap: 8px;
}
.s-row-control-tight {
  gap: 6px;
}

/* Toggle */
.s-toggle {
  appearance: none;
  border: 0;
  width: 36px;
  height: 20px;
  border-radius: 999px;
  background: var(--color-base-300);
  position: relative;
  cursor: pointer;
  transition: background 0.15s;
  flex-shrink: 0;
  padding: 0;
}
.s-toggle::after {
  content: "";
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--color-base-content);
  opacity: 0.7;
  transition: transform 0.15s;
}
.s-toggle.on {
  @apply bg-primary;
}
.s-toggle.on::after {
  transform: translateX(16px);
  @apply bg-primary-content;
  opacity: 1;
}

/* Segmented control */
.s-seg {
  display: inline-flex;
  @apply border-base-content/15 bg-base-100;
  border-width: 1px;
  border-radius: 7px;
  padding: 2px;
}
.s-seg button {
  appearance: none;
  background: transparent;
  border: 0;
  @apply text-base-content/60;
  font-family: inherit;
  font-size: 12px;
  font-weight: 500;
  padding: 4px 10px;
  border-radius: 5px;
  cursor: pointer;
}
.s-seg button.active {
  @apply bg-base-300 text-base-content border-base-content/15;
  border-width: 1px;
}

/* DropdownMenu visual override (it uses btn-primary by default) */
.s-row-control :deep(.dropdown > summary.btn) {
  @apply bg-base-100 text-base-content border-base-content/15 hover:bg-base-300;
  border-width: 1px;
  font-size: 12.5px;
  font-weight: 500;
  min-height: 0;
  height: auto;
  padding: 6px 10px;
  min-width: 96px;
  box-shadow: none;
}

/* Sponsor row */
.s-sponsor-row {
  flex-wrap: wrap;
  font-size: 12.5px;
  @apply text-base-content/60;
}
.s-sponsor-row a {
  @apply text-primary;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.s-sponsor-row a:hover {
  text-decoration: underline;
  text-underline-offset: 4px;
}
.s-sep {
  @apply text-base-content/40;
}

/* Display section: live preview alongside */
.s-display-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
  align-items: start;
}
@container (min-width: 56rem) {
  .s-display-grid {
    grid-template-columns: minmax(0, 1fr) minmax(320px, 40%);
  }
}
.settings-content {
  container-type: inline-size;
}

.s-preview {
  @apply border-base-content/15 bg-base-100;
  border-width: 1px;
  border-radius: 10px;
  overflow: hidden;
  position: sticky;
  top: 16px;
  display: none;
}
@container (min-width: 56rem) {
  .s-preview {
    display: block;
  }
}
</style>
