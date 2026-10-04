<template>
  <dialog ref="dialog" class="modal max-md:modal-bottom" @close="request = undefined">
    <div v-if="shown && target" class="modal-box flex max-w-lg flex-col gap-4">
      <div class="flex flex-col gap-1">
        <h3 class="text-2xl font-bold">{{ $t("rollback.title", { name: shown.name }) }}</h3>
        <p class="text-base-content/60 text-sm">
          <i18n-t keypath="rollback.body">
            <template #target>
              <span class="text-base-content font-mono">{{ rollbackLabel(target) }}</span>
            </template>
          </i18n-t>
        </p>
      </div>

      <InlineNotice v-if="holdsData(shown)" type="warning">{{ $t("rollback.volumes-warning") }}</InlineNotice>
      <InlineNotice v-if="composeManaged(shown)" type="warning">{{ $t("rollback.compose-warning") }}</InlineNotice>
      <p class="text-base-content/40 text-xs">{{ $t("rollback.schedule-note") }}</p>

      <div class="modal-action mt-0">
        <form method="dialog">
          <button class="btn btn-sm">{{ $t("button.cancel") }}</button>
        </form>
        <button class="btn btn-primary btn-sm" @click="confirm">{{ $t("rollback.confirm") }}</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import { Container, rollbackLabel } from "@/models/Container";

const request = useRollbackRequest();
const dialog = useTemplateRef<HTMLDialogElement>("dialog");

// Kept after the dialog closes: the rollback it started still reports on it.
const shown = shallowRef<Container>();
const target = computed(() => shown.value && rollbackTargetOf(shown.value));
const { rollback } = useContainerActions(toRef(() => shown.value as Container));

watch(request, (container) => {
  if (!container) return;
  shown.value = container;
  nextTick(() => dialog.value?.showModal());
});

function confirm() {
  const to = target.value?.imageId;
  dialog.value?.close();
  if (to) rollback(to);
}
</script>
