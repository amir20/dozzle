<template>
  <div class="flex items-center justify-end gap-4">
    <template v-if="config.pages">
      <router-link
        :to="{ name: 'content-id', params: { id: page.id } }"
        :title="page.title"
        v-for="page in config.pages"
        :key="page.id"
        class="link-primary"
      >
        {{ page.title }}
      </router-link>
    </template>
    <template v-if="config.user">
      <div v-if="config.authProvider === 'simple'">
        <button @click.prevent="logout()" class="link-primary">{{ $t("button.logout") }}</button>
      </div>
      <div>
        {{ config.user.name ? config.user.name : config.user.email }}
      </div>
      <img class="h-10 w-10 rounded-full p-1 ring-2 ring-base-content/50" :src="config.user.avatar" />
    </template>
  </div>
</template>
<script lang="ts" setup>
async function logout() {
  await fetch(withBase("/api/token"), {
    method: "DELETE",
  });

  location.reload();
}
</script>
