<template>
  <ul class="events" ref="events" :class="{ 'disable-wrap': !softWrap, [size]: true }">
    <li
      v-for="(item, index) in filtered"
      :key="item.id"
      :data-key="item.id"
      :class="{ selected: toRaw(item) === toRaw(lastSelectedItem) }"
    >
      <div class="line-options" v-show="isSearching()">
        <dropdown-menu :class="{ 'is-last': index === filtered.length - 1 }" class="is-top minimal">
          <a class="dropdown-item" @click="handleJumpLineSelected($event, item)" :href="`#${item.id}`">
            <div class="level is-justify-content-start">
              <div class="level-left">
                <div class="level-item">
                  <cil:find-in-page class="mr-4" />
                </div>
              </div>
              <div class="level-right">
                <div class="level-item">Jump to Context</div>
              </div>
            </div>
          </a>
        </dropdown-menu>
      </div>
      <component :is="item.getComponent()" :log-entry="item" :visible-keys="visibleKeys.value"></component>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { type ComputedRef, toRaw } from "vue";
import { useRouteHash } from "@vueuse/router";
import { Container } from "@/models/Container";
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

let visibleKeys = persistentVisibleKeys(inject("container") as ComputedRef<Container>);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages, resetSearch, isSearching } = useSearchFilter();

const { messages } = toRefs(props);
const visible = filteredPayload(messages);
const filtered = filteredMessages(visible);

const events = ref<HTMLElement>();
let lastSelectedItem: LogEntry<string | JSONObject> | undefined = $ref(undefined);

function handleJumpLineSelected(e: Event, item: LogEntry<string | JSONObject>) {
  lastSelectedItem = item;
  resetSearch();
}

const routeHash = useRouteHash();
watch(
  routeHash,
  (hash) => {
    document.querySelector(`[data-key="${hash.substring(1)}"]`)?.scrollIntoView({ block: "center" });
  },
  { immediate: true, flush: "post" }
);
</script>
<style scoped lang="scss">
.events {
  padding: 1em 0;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;

  & > li {
    display: flex;
    word-wrap: break-word;
    padding: 0.2em 1em;

    &:last-child {
      scroll-snap-align: end;
      scroll-margin-block-end: 5rem;
    }

    &:nth-child(odd) {
      background-color: rgba(125, 125, 125, 0.08);
    }

    &.selected {
      border: 1px var(--secondary-color) solid;
    }

    & > .line-options {
      display: flex;
      flex-direction: row-reverse;
      margin-right: 1em;
    }
  }

  &.small {
    font-size: 60%;
  }

  &.medium {
    font-size: 80%;
  }

  &.large {
    font-size: 120%;
  }
}
</style>
