<template>
  <div class="mt-2">
    <button type="button" class="btn btn-ghost btn-xs gap-1" @click="expanded = !expanded">
      <mdi:chevron-right class="transition-transform" :class="{ 'rotate-90': expanded }" />
      {{ $t("notifications.destination-form.variables") }}
    </button>

    <div v-if="expanded" class="border-base-content/15 mt-2 space-y-3 rounded-lg border p-3">
      <div v-for="group in groups" :key="group.scope">
        <div class="text-base-content/60 mb-1 text-xs">
          {{ $t(`notifications.destination-form.variables-${group.scope}`) }}
        </div>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="variable in group.variables"
            :key="variable.path"
            type="button"
            class="border-base-content/15 bg-base-content/5 text-base-content/70 hover:border-primary hover:text-primary cursor-pointer rounded-md border px-2 py-0.5 font-mono text-xs transition-colors"
            :title="variable.detail"
            @click="emit('insert', `{{ ${variable.path} }}`)"
          >
            {{ variable.path }}
          </button>
        </div>
      </div>
      <p class="text-base-content/50 text-xs">{{ $t("notifications.destination-form.variables-scope-hint") }}</p>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { TEMPLATE_VARIABLES, type TemplateVariable } from "@/composable/templateEditor";

const emit = defineEmits<{ insert: [snippet: string] }>();

const expanded = ref(false);

const groups = computed(() => {
  const scopes: TemplateVariable["scope"][] = ["all", "log", "metric", "event"];
  return scopes
    .map((scope) => ({ scope, variables: TEMPLATE_VARIABLES.filter((v) => v.scope === scope) }))
    .filter((g) => g.variables.length);
});
</script>
