<template>
  <div class="flex items-center justify-end gap-4">
    <slot name="more-items"></slot>
    <Announcements />

    <router-link
      v-if="config.enableNotifications"
      :to="{ name: '/notifications' }"
      :aria-label="$t('title.notifications')"
      data-testid="notifications"
      class="btn btn-circle btn-sm"
    >
      <mdi:bell class="icon-ring size-6" />
    </router-link>

    <CloudPopover v-if="config.enableCloud" />

    <router-link
      :to="{ name: '/settings' }"
      :aria-label="$t('title.settings')"
      data-testid="settings"
      class="btn btn-circle btn-sm"
    >
      <mdi:cog class="icon-spin size-6" />
    </router-link>

    <dropdown class="dropdown-end" data-testid="user-menu" v-if="config.user">
      <template #trigger>
        <template v-if="config.disableAvatars || !config.user.email">
          <material-symbols:person class="size-6" />
        </template>
        <template v-else>
          <img
            class="ring-base-content/25 size-6 max-w-none rounded-full p-px ring-1"
            :src="withBase('/api/profile/avatar')"
          />
        </template>
      </template>
      <template #content>
        <div class="max-w-72">
          <div class="flex items-center gap-3 px-2 py-1.5">
            <img
              v-if="!config.disableAvatars && config.user.email"
              class="ring-base-content/15 size-9 shrink-0 rounded-full ring-1"
              :src="withBase('/api/profile/avatar')"
            />
            <!-- min-w-0 is what lets a long email truncate instead of widening the panel. -->
            <div class="min-w-0">
              <div class="truncate text-sm font-semibold">{{ config.user.name }}</div>
              <div v-if="config.user.email" class="text-base-content/60 truncate text-xs">
                {{ config.user.email }}
              </div>
            </div>
          </div>

          <template v-if="config.authProvider === 'simple' || config.logoutUrl">
            <div class="bg-base-content/10 my-1.5 h-px"></div>
            <button
              @click.prevent="logout()"
              class="hover:bg-base-300 flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
            >
              <material-symbols:logout class="size-4 opacity-60" />
              {{ $t("button.logout") }}
            </button>
          </template>
        </div>
      </template>
    </dropdown>
  </div>
</template>
<script lang="ts" setup>
const { logoutUrl } = config;

async function logout() {
  if (logoutUrl) {
    location.href = logoutUrl;
  } else {
    await fetch(withBase("/api/token"), {
      method: "DELETE",
    });

    location.reload();
  }
}
</script>
