<template>
  <ul class="menu w-full p-0 text-[0.95rem]">
    <li v-if="customGroups.length > 0">
      <details open>
        <summary class="text-base-content/80 font-light">
          <ph:bounding-box-fill />
          {{ $t("label.group-menu") }}
        </summary>
        <ul>
          <li v-for="group in customGroups" :key="group.name">
            <router-link :to="{ name: '/group/[name]', params: { name: group.name } }" active-class="menu-active">
              <ph:stack-simple />
              <div class="truncate">
                {{ group.name }}
              </div>
            </router-link>
          </li>
        </ul>
      </details>
    </li>
  </ul>
</template>

<script lang="ts" setup>
const store = useSwarmStore();

const { customGroups } = storeToRefs(store);
</script>
<style scoped></style>
