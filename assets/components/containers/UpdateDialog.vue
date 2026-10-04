<template>
  <dialog ref="dialog" class="modal max-md:modal-bottom" @close="request = undefined">
    <div v-if="shown" class="modal-box flex max-w-lg flex-col gap-4">
      <div class="flex flex-col gap-1">
        <h3 class="text-2xl font-bold">{{ $t("updates.confirm-title", { name: shown.name }) }}</h3>
        <p class="text-base-content/60 text-sm">
          <i18n-t keypath="updates.confirm-body">
            <template #image>
              <span class="text-base-content font-mono wrap-anywhere">{{ shown.image }}</span>
            </template>
          </i18n-t>
        </p>
      </div>

      <UpdateWatchCheckbox v-model="watchInCloud" />

      <div class="modal-action mt-0">
        <form method="dialog">
          <button class="btn btn-sm">{{ $t("button.cancel") }}</button>
        </form>
        <button class="btn btn-primary btn-sm" @click="confirm">{{ $t("toolbar.update") }}</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const request = useUpdateRequest();
const dialog = useTemplateRef<HTMLDialogElement>("dialog");

// Kept after the dialog closes: the update it started still reports on it.
const shown = shallowRef<Container>();
const watchInCloud = ref(true);
const { update } = useContainerActions(toRef(() => shown.value as Container));

watch(request, (container) => {
  if (!container) return;
  shown.value = container;
  watchInCloud.value = true;
  nextTick(() => dialog.value?.showModal());
});

function confirm() {
  dialog.value?.close();
  update({ watchInCloud: watchInCloud.value });
}
</script>
