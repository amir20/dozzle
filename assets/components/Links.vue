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

    <dropdown class="dropdown-end" @closed="latestTag = latest?.tag ?? config.version">
      <template #trigger>
        <mdi:announcement class="h-6 w-6 -rotate-12" />
        <span
          class="absolute right-px top-0 h-2 w-2 rounded-full bg-red"
          v-if="hasUpdate && latestTag != latest?.tag"
        ></span>
      </template>
      <template #content>
        <div class="w-72">
          <releases />
        </div>
      </template>
    </dropdown>

    <dropdown class="dropdown-end" v-if="config.user">
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
    </dropdown>
  </div>
</template>
<script lang="ts" setup>
async function logout() {
  await fetch(withBase("/api/token"), {
    method: "DELETE",
  });

  location.reload();
}

const { hasUpdate, latest } = useReleases();
const latestTag = useStorage("DOZZLE_LATEST_TAG", config.version);
</script>
