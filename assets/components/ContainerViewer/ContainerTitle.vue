<template>
  <div class="@container flex min-w-0 flex-1 items-center gap-1.5 md:gap-2">
    <label class="icon-btn swap swap-rotate size-4">
      <input type="checkbox" v-model="pinned" />
      <carbon:star-filled class="swap-on text-secondary" />
      <carbon:star class="swap-off" />
    </label>
    <div class="inline-flex min-w-0 items-center text-sm">
      <!-- daisyUI insets breadcrumbs with a -0.25rem margin plus a matching
           0.25rem list padding. That pair is not accounted for in the intrinsic
           width, so the truncating name below always came up 4px short and
           ellipsized even with the whole row free. Zeroing both keeps the text
           at the same x and gives it back those 4px. -->
      <div class="breadcrumbs ms-0 min-w-0 overflow-x-visible p-0 font-mono">
        <ul class="ps-0">
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
                  {{ container.name }}
                  <span class="badge badge-xs badge-neutral font-sans">{{ sameNameContainers.length }}</span>
                  <carbon:caret-down />
                </button>
                <ul
                  tabindex="0"
                  class="dropdown-content menu rounded-box bg-base-100 border-base-content/20 z-10 w-max border p-1 shadow-sm"
                >
                  <li class="menu-title px-2 py-1 text-xs">
                    {{ $t("label.container", sameNameContainers.length) }}
                  </li>
                  <li v-for="other in sameNameContainers" :key="other.id">
                    <router-link
                      :to="{ name: '/container/[id]', params: { id: other.id } }"
                      active-class="menu-active"
                      class="grid grid-cols-[auto_1fr_auto] items-center gap-x-3"
                      :title="other.isSwarm ? other.swarmId : other.name"
                    >
                      <div
                        class="status data-[state=exited]:status-error data-[state=running]:status-success data-[state=paused]:status-warning"
                        :data-state="other.state"
                      ></div>
                      <div class="flex flex-col leading-tight">
                        <span class="font-mono text-xs">{{ other.id.slice(0, 12) }}</span>
                        <span v-if="showHost" class="text-base-content/50 font-sans text-[11px]">
                          {{ other.hostLabel }}
                        </span>
                      </div>
                      <div class="flex flex-col text-right font-sans leading-tight">
                        <span class="text-xs">{{ other.state }}</span>
                        <RelativeTime :date="timestampOf(other)" class="text-base-content/50 text-[11px]" />
                      </div>
                    </router-link>
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
    <ContainerLink :container="container" />
    <ContainerLinkHint :container="container" />
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

const sameNameContainers = computed(() =>
  allContainers.value
    .filter((c) => c.name === container.name && c.customGroup === container.customGroup)
    .sort((a, b) => +b.created - +a.created),
);

const otherContainers = computed(() => sameNameContainers.value.filter((c) => c.id !== container.id));

// Same-name containers are usually the same service restarted, so the host only tells them
// apart when they actually differ.
const showHost = computed(() => new Set(sameNameContainers.value.map((c) => c.hostLabel)).size > 1);

// Docker leaves startedAt/finishedAt at the zero time when a container never ran or is still
// running, which renders as "2027 years ago". Fall back to the one timestamp always set.
const isSet = (date: Date) => date.getFullYear() > 1;

function timestampOf(c: Container) {
  const date = c.state === "running" || c.state === "paused" ? c.startedAt : c.finishedAt;
  return isSet(date) ? date : c.created;
}
</script>

<style scoped></style>
