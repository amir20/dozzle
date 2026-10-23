<template>
  <Popover placement="bottom-end" panel-class="log-menu-panel" :close-on-select="false">
    <template #trigger>
      <button
        type="button"
        class="icon-btn btn btn-ghost btn-xs"
        :title="$t('label.columns')"
        :aria-label="$t('label.columns')"
      >
        <ph:columns />
      </button>
    </template>
    <ul class="menu log-menu w-full p-0">
      <li class="section">{{ $t("label.columns") }}</li>
      <li v-for="column in columns" :key="column.key">
        <a @click="toggle(column.key)">
          <mdi:check class="w-4" v-if="!hidden.includes(column.key)" />
          <div v-else class="w-4"></div>
          {{ $t(column.label) }}
        </a>
      </li>
    </ul>
    <div class="bg-base-content/10 my-1 h-px"></div>
    <ul class="menu log-menu w-full p-0">
      <li>
        <a @click="average = !average">
          <mdi:check class="w-4" v-if="average" />
          <div v-else class="w-4"></div>
          {{ $t("label.moving-average") }}
        </a>
      </li>
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
const { columns } = defineProps<{
  columns: { key: string; label: string }[];
}>();

const hidden = defineModel<string[]>("hidden", { required: true });
const average = defineModel<boolean>("average", { required: true });

function toggle(key: string) {
  hidden.value = hidden.value.includes(key) ? hidden.value.filter((k) => k !== key) : [...hidden.value, key];
}
</script>
