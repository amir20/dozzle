<template>
  <ul class="menu p-0 text-[0.95rem]">
    <li v-for="{ name, services } in stacks" :key="name">
      <details open>
        <summary class="font-light text-base-content/80">
          <ph:stack />
          {{ name }}
        </summary>
        <ul>
          <li v-for="service in services" :key="service.name">
            <router-link :to="{ name: 'service-name', params: { name: service.name } }" active-class="active-primary">
              <ph:stack-simple />
              <div class="truncate">
                {{ service.name }}
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

const { stacks } = storeToRefs(store);
</script>
<style scoped lang="postcss"></style>
