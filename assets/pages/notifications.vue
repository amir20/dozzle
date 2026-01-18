<template>
  <PageWithLinks>
    <section>
      <div class="mb-6">
        <h2 class="text-2xl font-bold">Notifications</h2>
        <p class="text-base-content/60">Manage your log event subscriptions</p>
      </div>

      <div class="divider"></div>

      <button class="btn btn-primary" @click="showDrawer(CreateAlert, {}, 'lg')">
        <mdi:plus />
        Create Alert
      </button>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import CreateAlert from "@/components/Notification/CreateAlert.vue";

const showDrawer = useDrawer();
</script>
