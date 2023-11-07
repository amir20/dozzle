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
      <div class="dropdown dropdown-end">
        <label tabindex="0" class="btn btn-circle btn-sm">
          <img class="h-10 w-10 max-w-none rounded-full p-1 ring-2 ring-base-content/50" :src="config.user.avatar" />
        </label>
        <div
          tabindex="0"
          class="dropdown-content rounded-box z-50 mt-1 w-52 border border-base-content/20 bg-base p-2 shadow"
        >
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
        </div>
      </div>
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
