<template>
  <div class="flex items-center">
    <div class="breadcrumbs flex-1">
      <ul>
        <li>
          <a @click.prevent="setNamespace(null)" class="link-primary">{{ $t("label.namespaces") }}</a>
        </li>
        <li v-if="selectedNamespace === 'all'">
          {{ $t("label.all-namespaces") }}
        </li>
        <li v-else-if="selectedNamespace" class="cursor-default">
          <router-link
            :to="{
              name: '/namespace/[name]',
              params: { name: selectedNamespace },
            }"
            class="btn btn-outline btn-primary btn-xs"
            active-class="btn-active"
          >
            <ph:arrows-merge />
            {{ selectedNamespace }}
          </router-link>
        </li>
      </ul>
    </div>
    <div class="flex-none">
      <div class="dropdown dropdown-end dropdown-hover">
        <label tabindex="0" class="btn btn-square btn-ghost btn-sm">
          <ph:dots-three-vertical-bold />
        </label>
        <ul
          tabindex="0"
          class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
        >
          <li>
            <a class="text-sm capitalize" @click="collapseAll()">
              <material-symbols-light:collapse-all class="w-4" />
              {{ $t("label.collapse-all") }}
            </a>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <SlideTransition :slide-right="selectedNamespace !== null">
    <template #left>
      <ul class="menu p-0">
        <li>
          <a @click.prevent="setNamespace('all')">
            <ph:circles-four />
            {{ $t("label.all-namespaces") }}
          </a>
        </li>
        <li v-for="ns in namespaces" :key="ns.name">
          <a @click.prevent="setNamespace(ns.name)">
            <ph:circles-four />
            {{ ns.name }}
          </a>
        </li>
      </ul>
    </template>
    <template #right>
      <ul class="menu w-full p-0 text-[0.95rem]" ref="menu">
        <li v-for="{ name, owners } in filteredNamespaces" :key="name">
          <details open>
            <summary class="text-base-content/80 font-light">
              <ph:stack />
              {{ name }} ({{ owners.length }})

              <router-link
                :to="{ name: '/namespace/[name]', params: { name } }"
                class="btn btn-square btn-outline btn-primary btn-xs"
                active-class="btn-active"
                :title="$t('tooltip.merge-owners')"
              >
                <ph:arrows-merge />
              </router-link>
            </summary>
            <ul>
              <li v-for="owner in owners" :key="`${owner.kind}-${owner.name}`">
                <router-link :to="{ name: '/owner/[name]', params: { name: owner.name } }" active-class="menu-active">
                  <ph:stack-simple />
                  <div class="truncate">{{ owner.kind }}/{{ owner.name }}</div>
                </router-link>
              </li>
            </ul>
          </details>
        </li>

        <li v-if="ownersWithoutNamespace.length > 0">
          <details open>
            <summary class="text-base-content/80 font-light">
              <ph:circles-four />
              {{ $t("label.owners") }} ({{ ownersWithoutNamespace.length }})
            </summary>
            <ul>
              <li v-for="owner in ownersWithoutNamespace" :key="`${owner.kind}-${owner.name}`">
                <router-link :to="{ name: '/owner/[name]', params: { name: owner.name } }" active-class="menu-active">
                  <ph:stack-simple />
                  <div class="truncate">{{ owner.kind }}/{{ owner.name }}</div>
                </router-link>
              </li>
            </ul>
          </details>
        </li>
      </ul>
    </template>
  </SlideTransition>
</template>

<script lang="ts" setup>
const store = useK8sStore();

const { namespaces, owners } = storeToRefs(store);

const selectedNamespace = ref<string | null>("all");

const setNamespace = (namespace: string | null) => (selectedNamespace.value = namespace);

const filteredNamespaces = computed(() => {
  if (selectedNamespace.value === null || selectedNamespace.value === "all") {
    return namespaces.value;
  }
  return namespaces.value.filter((ns) => ns.name === selectedNamespace.value);
});

const ownersWithoutNamespace = computed(() => {
  const filtered = owners.value.filter((owner) => !owner.namespace);
  if (selectedNamespace.value === null || selectedNamespace.value === "all") {
    return filtered;
  }
  return [];
});

const menu = useTemplateRef("menu");

const collapseAll = () => {
  const details = menu.value?.querySelectorAll("details");
  details?.forEach((detail) => (detail.open = false));
};
</script>
