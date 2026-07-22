<template>
  <div class="flex flex-col gap-8 px-4 py-4 md:px-8">
    <section>
      <Links>
        <template #more-items>
          <Tag class="font-mono">{{ config.version }}</Tag>
        </template>
      </Links>
    </section>

    <SettingsPanel />
  </div>
</template>

<script lang="ts" setup>
import SettingsPanel from "@/components/Settings/SettingsPanel.vue";

const { t } = useI18n();
setTitle(t("title.settings"));
</script>
