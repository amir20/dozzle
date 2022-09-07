<template>
  <div>
    <section class="hero is-small mt-4">
      <div class="hero-body">
        <div class="container">
          <div class="columns">
            <div class="column">
              <h1 class="title">Hello, there!</h1>
            </div>
            <div class="column is-narrow" v-if="secured">
              <a class="button is-primary is-small" :href="`${base}/logout`">Logout</a>
            </div>
          </div>
        </div>
      </div>
    </section>
    <section class="level section">
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ containers.length }}</p>
          <p class="heading">Total Containers</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ runningContainers.length }}</p>
          <p class="heading">Running</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title" data-ci-skip>{{ totalCpu }}%</p>
          <p class="heading">Total CPU Usage</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title" data-ci-skip>{{ formatBytes(totalMem) }}</p>
          <p class="heading">Total Mem Usage</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ version }}</p>
          <p class="heading">Dozzle Version</p>
        </div>
      </div>
    </section>

    <section class="columns is-centered section is-marginless">
      <div class="column is-4">
        <div class="panel">
          <p class="panel-heading">Containers</p>
          <div class="panel-block">
            <p class="control has-icons-left">
              <input
                class="input"
                type="text"
                placeholder="Search Containers"
                v-model="search"
                @keyup.esc="search = null"
                @keyup.enter="onEnter()"
              />
              <span class="icon is-left">
                <search-icon />
              </span>
            </p>
          </div>
          <p class="panel-tabs" v-if="!search">
            <a :class="{ 'is-active': sort === 'running' }" @click="sort = 'running'">Running</a>
            <a :class="{ 'is-active': sort === 'all' }" @click="sort = 'all'">All</a>
          </p>
          <router-link
            :to="{ name: 'container', params: { id: item.id } }"
            v-for="item in results.slice(0, 10)"
            :key="item.id"
            class="panel-block"
          >
            <span class="name">{{ item.name }}</span>

            <div class="subtitle is-7 status">
              <past-time :date="new Date(item.created * 1000)"></past-time>
            </div>
          </router-link>
        </div>
      </div>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { storeToRefs } from "pinia";
import { useRouter } from "vue-router";
import { useContainerStore } from "@/stores/container";
import { formatBytes } from "@/utils";
import fuzzysort from "fuzzysort";
import SearchIcon from "~icons/mdi-light/magnify";
import PastTime from "../components/PastTime.vue";
import config from "@/stores/config";
import { useIntervalFn } from "@vueuse/core";

const { base, version, secured } = config;
const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore);
const router = useRouter();

const sort = ref("running");
const search = ref();

const results = computed(() => {
  if (search.value) {
    return fuzzysort.go(search.value, containers.value, { key: "name" }).map((i) => i.obj);
  }
  switch (sort.value) {
    case "all":
      return mostRecentContainers.value;
    case "running":
      return runningContainers.value;
    default:
      throw `Invalid sort order: ${sort.value}`;
  }
});

const mostRecentContainers = computed(() => [...containers.value].sort((a, b) => b.created - a.created));
const runningContainers = computed(() => mostRecentContainers.value.filter((c) => c.state === "running"));
const totalCpu = ref(0);
useIntervalFn(
  () => {
    totalCpu.value = runningContainers.value.reduce((acc, c) => acc + (c.stat?.cpu ?? 0), 0);
  },
  1000,
  { immediate: true }
);
const totalMem = ref(0);

useIntervalFn(
  () => {
    totalMem.value = runningContainers.value.reduce((acc, c) => acc + (c.stat?.memoryUsage ?? 0), 0);
  },
  1000,
  { immediate: true }
);

function onEnter() {
  if (results.value.length == 1) {
    const [item] = results.value;
    router.push({ name: "container", params: { id: item.id } });
  }
}
</script>
<style lang="scss" scoped>
.panel {
  border: 1px solid var(--border-color);
  .panel-block,
  .panel-tabs {
    border-color: var(--border-color);
    .is-active {
      border-color: var(--border-hover-color);
    }
    .name {
      text-overflow: ellipsis;
      white-space: nowrap;
      overflow: hidden;
    }
    .status {
      margin-left: auto;
      white-space: nowrap;
    }
  }
}

.icon {
  padding: 10px 3px;
}
</style>
