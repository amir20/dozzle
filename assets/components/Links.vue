<template>
  <div class="flex items-center justify-end gap-4">
    <slot name="more-items"></slot>
    <Announcements />

    <router-link
      :to="{ name: '/settings' }"
      :aria-label="$t('title.settings')"
      data-testid="settings"
      class="btn btn-circle btn-sm"
    >
      <mdi:cog class="size-6" />
    </router-link>

    <dropdown class="dropdown-end" v-if="config.user">
      <template #trigger>
        <img
          class="ring-base-content/60 size-6 max-w-none rounded-full p-px ring-1"
          :src="withBase('/api/profile/avatar')"
        />
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
            <button @click.prevent="logout()" class="text-primary p-2">
              <material-symbols:logout />
              {{ $t("button.logout") }}
            </button>
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
</script>
