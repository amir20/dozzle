<template>
  <div class="@container flex min-w-0 flex-1 items-center gap-1.5 md:gap-2">
    <label class="swap swap-rotate size-4">
      <input type="checkbox" v-model="pinned" />
      <carbon:star-filled class="swap-on text-secondary" />
      <carbon:star class="swap-off" />
    </label>
    <div class="inline-flex min-w-0 items-center text-sm">
      <div class="breadcrumbs min-w-0 overflow-x-visible p-0 font-mono">
        <ul>
          <li v-if="config.hosts.length > 1" class="font-thin max-md:hidden">
            {{ container.hostLabel }}
          </li>
          <li class="min-w-0">
            <template v-if="otherContainers.length === 0"
              ><span class="block truncate">{{ container.name }}</span></template
            >
            <div v-else>
              <div class="dropdown">
                <button tabindex="0" role="button" class="btn btn-xs md:btn-sm">
                  {{ container.name }} <carbon:caret-down />
                </button>
                <ul
                  tabindex="0"
                  class="dropdown-content menu rounded-box bg-base-100 border-base-content/20 border shadow-sm"
                >
                  <li v-for="other in otherContainers">
                    <router-link :to="{ name: '/container/[id]', params: { id: other.id } }">
                      <div
                        class="status data-[state=exited]:status-error data-[state=running]:status-success data-[state=paused]:status-warning"
                        :data-state="other.state"
                      ></div>
                      <div v-if="other.isSwarm">{{ other.swarmId }}</div>
                      <div v-else>{{ other.name }}</div>
                      <div v-if="other.state === 'running'">running</div>
                      <div v-else-if="other.state === 'paused'">paused</div>
                      <RelativeTime :date="other.finishedAt" class="text-base-content/70 text-xs" v-else />
                    </router-link>
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
    <ContainerHealth :health="container.health" v-if="container.health" />
    <VolumeWarning :container="container" />
    <Tag
      class="group hidden! cursor-pointer items-center gap-1.5 pr-1! font-mono @md:inline-flex!"
      size="small"
      role="button"
      :title="$t('toolbar.copy-image')"
      :aria-label="$t('toolbar.copy-image')"
      @click="copyImage"
    >
      <span class="truncate">{{ imageTag }}</span>
      <span
        class="bg-base-content/10 text-base-content/40 group-hover:text-base-content/70 flex size-4 shrink-0 items-center justify-center rounded-sm transition-colors"
      >
        <mdi:content-copy class="size-3" />
      </span>
    </Tag>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();

const { t } = useI18n();
const { copy, copied, isSupported } = useClipboard({ legacy: true });
const { showToast } = useToast();

const imageTag = computed(() => container.image.replace(/@sha.*/, ""));

async function copyImage() {
  if (!isSupported.value) return;
  await copy(imageTag.value);
  if (copied.value) {
    showToast({ title: t("toasts.copied.title"), message: t("toasts.copied.message"), type: "info" }, { expire: 2000 });
  }
}

const pinned = computed({
  get: () => pinnedContainers.value.has(container.name),
  set: (value) => {
    if (value) {
      pinnedContainers.value.add(container.name);
    } else {
      pinnedContainers.value.delete(container.name);
    }
  },
});
const store = useContainerStore();
const { containers: allContainers } = storeToRefs(store);

const otherContainers = computed(() =>
  allContainers.value
    .filter((c) => c.name === container.name && c.id !== container.id)
    .sort((a, b) => +b.created - +a.created),
);
</script>

<style scoped></style>
