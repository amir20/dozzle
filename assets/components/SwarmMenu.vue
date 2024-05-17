<template>
  <ul class="containers menu p-0 [&_li.menu-title]:px-0">
    <li v-for="{ name, services } in stacks" :key="name">
      <details open>
        <summary class="font-light text-base-content/80">
          <ph:stack />
          {{ name }}
        </summary>
        <ul>
          <li v-for="service in services" :key="service.name">
            {{ service.name }}
          </li>
        </ul>
      </details>
    </li>
  </ul>
</template>

<script lang="ts" setup>
const store = useSwarmStore();

const { stacks } = storeToRefs(store);
</script>
<style scoped lang="postcss">
.menu {
  @apply text-[0.95rem];
}
.containers a {
  @apply auto-cols-[auto_max-content_max-content];
  .pin {
    display: none;

    &:hover {
      @apply text-secondary;
    }
  }

  &:hover {
    .pin {
      display: inline-block;
    }
  }
}
li.exited {
  @apply opacity-50;
}
</style>
