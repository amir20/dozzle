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

    <icon-dropdown class="dropdown-end">
      <template #trigger>
        <mdi:announcement class="h-6 w-6 -rotate-12" />
      </template>
      <template #content>
        <ul>
          <li v-for="release in releases">
            {{ release.name }}
          </li>
        </ul>
      </template>
    </icon-dropdown>

    <icon-dropdown class="dropdown-end" v-if="config.user">
      <template #trigger>
        <img class="h-8 w-8 max-w-none rounded-full p-1 ring-2 ring-base-content/50" :src="config.user.avatar" />
      </template>
      <template #content>
        <div class="p-2">
          <div class="font-bold">
            {{ config.user.name }}
          </div>
          <div class="text-sm font-light">
            {{ config.user.email }}
          </div>
        </div>
        <ul class="menu mt-4 p-0">
          <li v-if="config.authProvider === 'simple'">
            <button @click.prevent="logout()" class="text-primary">{{ $t("button.logout") }}</button>
          </li>
        </ul>
      </template>
    </icon-dropdown>
  </div>
</template>
<script lang="ts" setup>
async function logout() {
  await fetch(withBase("/api/token"), {
    method: "DELETE",
  });

  location.reload();
}

const { hasUpdate, releases } = useReleases();
</script>
