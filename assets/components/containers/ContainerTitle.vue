<template>
  <div class="@container flex min-w-0 flex-1 items-center gap-1.5 md:gap-2">
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
            <div v-else class="min-w-0">
              <!-- The anchor is inline-block, so it sizes to its content and silently
                   overflows this shrunken li: the button spilled past the row and the next
                   control painted over its caret. max-w-full puts it back under the li's
                   width so the name truncates instead. -->
              <Popover
                class="max-w-full min-w-0"
                panel-class="rounded-box bg-base-100 border-base-content/20 w-max border p-1 shadow-sm"
              >
                <template #trigger>
                  <!-- Ghost until hover: the name is the page title, and a
                       permanent button frame made it read as one more chip. -->
                  <button type="button" class="btn btn-ghost btn-xs md:btn-sm max-w-full min-w-0 px-1.5">
                    <span class="truncate">{{ container.name }}</span>
                    <!-- The count and caret are the only affordance for the menu, so they
                         must survive a long image tag squeezing this button. -->
                    <span class="badge badge-xs badge-neutral shrink-0 font-sans">{{ sameNameContainers.length }}</span>
                    <carbon:caret-down class="text-base-content/50 shrink-0" />
                  </button>
                </template>
                <ul class="menu w-full p-0">
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
              </Popover>
            </div>
          </li>
        </ul>
      </div>
    </div>
    <!-- Sits after the name so the name leads the row, and carries the same map-pin the
         sidebar's "Pinned" section uses: a star here and a pin there read as two unrelated
         features, which is why people never found this. The word "Pin" would cost too much
         of the row, so the labelled copy of this lives in the toolbar menu. -->
    <button
      class="icon-btn shrink-0"
      :class="pinned ? 'text-secondary' : 'text-base-content/40 hover:text-base-content'"
      :aria-pressed="pinned"
      :title="pinned ? $t('toolbar.unpin') : $t('toolbar.pin')"
      :aria-label="pinned ? $t('toolbar.unpin') : $t('toolbar.pin')"
      @click="pinned = !pinned"
    >
      <ph:map-pin-simple-fill v-if="pinned" class="size-4" />
      <ph:map-pin-simple v-else class="size-4" />
    </button>

    <ContainerLink :container="container" />
    <ContainerLinkHint :container="container" />
    <ContainerHealth :health="container.health" v-if="container.health" />
    <VolumeWarning :container="container" />
    <!-- The image is reference material, not a control: plain dimmed text keeps
         it out of the name's way, and the copy affordance appears on hover. -->
    <button
      class="group text-base-content/45 hover:text-base-content/80 hidden max-w-[32ch] min-w-0 cursor-copy items-center gap-1.5 font-mono text-xs transition-colors @md:inline-flex"
      :title="$t('toolbar.copy-image')"
      :aria-label="$t('toolbar.copy-image')"
      @click="copyImage"
    >
      <span class="truncate">{{ imageTag }}</span>
      <mdi:content-copy class="size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();

const { copy } = useCopy();

const imageTag = computed(() => container.image.replace(/@sha.*/, ""));

async function copyImage() {
  await copy(imageTag.value);
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
