<template>
  <div class="mb-2 flex items-center">
    <div class="flex-1">
      {{ $t("label.owner", owners.length) }}
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
  <ul class="menu w-full p-0 text-[0.95rem]" ref="menu">
    <li v-for="{ name, owners } in namespaces" :key="name">
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

<script lang="ts" setup>
const store = useK8sStore();

const { namespaces, owners } = storeToRefs(store);

const ownersWithoutNamespace = computed(() => owners.value.filter((owner) => !owner.namespace));

const menu = useTemplateRef("menu");

const collapseAll = () => {
  const details = menu.value?.querySelectorAll("details");
  details?.forEach((detail) => (detail.open = false));
};
</script>
