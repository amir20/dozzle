<template>
  <!-- SIDEBAR -->
  <SettingsSection :title="$t('settings.sidebar')">
    <div
      class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y overflow-hidden rounded-lg border"
    >
      <SettingRow tag="label" :label="$t('settings.show-stopped-containers')" class="px-4">
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showAllContainers" />
      </SettingRow>
      <SettingRow
        :label="$t('settings.group-containers')"
        :description="$t('settings.group-containers-desc')"
        class="px-4"
      >
        <DropdownMenu
          plain
          v-model="groupContainers"
          :options="[
            { label: $t('settings.grouping.always'), value: 'always' },
            { label: $t('settings.grouping.at-least-2'), value: 'at-least-2' },
            { label: $t('settings.grouping.never'), value: 'never' },
          ]"
        />
      </SettingRow>
      <SettingRow
        tag="label"
        :label="$t('settings.show-app-icons')"
        :description="$t('settings.show-app-icons-desc')"
        class="px-4"
      >
        <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showAppIcons" />
      </SettingRow>
    </div>
  </SettingsSection>
</template>

<script lang="ts" setup>
import { groupContainers, showAllContainers, showAppIcons } from "@/stores/settings";
</script>
