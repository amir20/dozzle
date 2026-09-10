<template>
  <!--
    What the assistant is looking at, made visible.

    If it silently knows what is on your screen you cannot tell what it knows,
    and one wrong assumption makes you distrust all of it. Showing the context
    as chips also answers "does the pane survive navigation" as something the
    reader watches happen rather than a rule they have to trust.
  -->
  <div class="flex flex-wrap items-center gap-1.5">
    <span v-for="chip in chips" :key="chip.key" class="status-pill status-pill-neutral gap-1">
      {{ chip.label }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import type { ViewContext } from "@/composable/viewContext";

const { view } = defineProps<{ view: ViewContext }>();
const { t } = useI18n();

const chips = computed(() => {
  const out: { key: string; label: string }[] = [];
  const names = view.containers.map((c) => c.name);

  if (names.length === 1) {
    out.push({ key: "container", label: names[0] });
  } else if (names.length > 1) {
    out.push({ key: "container", label: t("cloud-chat.n-containers", { n: names.length }) });
  } else if (view.target) {
    out.push({ key: "target", label: view.target });
  }

  if (view.search) out.push({ key: "search", label: `"${view.search}"` });
  if (view.levels?.length) out.push({ key: "levels", label: view.levels.join(" · ") });
  if (view.historical) out.push({ key: "historical", label: t("cloud-chat.historical") });

  return out;
});
</script>
