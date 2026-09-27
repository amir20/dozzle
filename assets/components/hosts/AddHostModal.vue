<template>
  <dialog ref="dialog" class="modal" @cancel="onCancel" @close="isOpen = false">
    <div class="modal-box flex max-h-[90vh] w-full max-w-2xl flex-col p-0">
      <div class="overflow-y-auto p-6">
        <h2 class="text-2xl font-bold">{{ $t("setup.hosts.title") }}</h2>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.hosts.subtitle") }}</p>

        <template v-if="isOpen">
          <AddHostPanel v-if="status" ref="panel" :status="status" class="mt-6" />
          <div v-else-if="loading" class="flex justify-center py-10">
            <span class="loading loading-spinner loading-sm"></span>
          </div>
          <InlineNotice v-else type="error" class="mt-6">{{ $t("setup.error.load") }}</InlineNotice>
        </template>
      </div>
      <div class="border-base-content/10 flex justify-end border-t px-6 py-4">
        <button type="button" class="btn btn-sm" :disabled="busy" @click="close">{{ $t("setup.close") }}</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button :disabled="busy">{{ $t("setup.close") }}</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import type AddHostPanel from "@/components/hosts/AddHostPanel.vue";

const { status, loading, fetchStatus } = useSetup();

const dialog = useTemplateRef<HTMLDialogElement>("dialog");
const panel = useTemplateRef<InstanceType<typeof AddHostPanel>>("panel");
const busy = computed(() => !!panel.value?.busy);
// The panel mounts only while open: mounting creates the private agent pair in
// /data, and closing resets the form for the next open.
const isOpen = ref(false);

// The status is shared with the wizard, so it may already be here. Read it again
// anyway: the list of agents is what this dialog is about.
function open() {
  isOpen.value = true;
  dialog.value?.showModal();
  fetchStatus();
}

function close() {
  dialog.value?.close();
}

// Escape mid-request would hide whether the host was added.
function onCancel(e: Event) {
  if (busy.value) e.preventDefault();
}

defineExpose({ open, close });
</script>
