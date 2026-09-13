<template>
  <div class="flex items-center justify-end gap-4">
    <slot name="more-items"></slot>
    <Announcements />

    <router-link
      v-if="config.enableNotifications"
      :to="{ name: '/notifications' }"
      :aria-label="unseenAlerts ? $t('notifications.new-alerts') : $t('title.notifications')"
      :title="unseenAlerts ? $t('notifications.new-alerts') : undefined"
      data-testid="notifications"
      class="btn btn-circle btn-sm relative"
    >
      <mdi:bell class="icon-ring size-6" />
      <!-- Severity rides the dot, the button behind it stays neutral. Same mark
           the container rows carry, so one glance across the app means the same
           thing everywhere. -->
      <span v-if="unseenAlerts" class="absolute end-1 top-1 flex size-1.5">
        <span class="bg-warning absolute size-full rounded-full opacity-75 motion-safe:animate-ping"></span>
        <span class="bg-warning relative size-full rounded-full"></span>
      </span>
    </router-link>

    <CloudPopover v-if="cloudSurfaceMounted" />

    <router-link
      :to="{ name: '/settings' }"
      :aria-label="$t('title.settings')"
      data-testid="settings"
      class="btn btn-circle btn-sm"
    >
      <mdi:cog class="icon-spin size-6" />
    </router-link>

    <Dropdown placement="bottom-end" data-testid="user-menu" v-if="config.user">
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

          <template v-if="hasSession || config.logoutUrl">
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
    </Dropdown>
  </div>
</template>
<script lang="ts" setup>
const { mounted: cloudSurfaceMounted } = useCloudSurface();
const { logoutUrl } = config;

// simple and oidc hold a session cookie Dozzle issued, so logging out means
// clearing it. Forward proxy has none: the logout URL is the whole logout there.
const hasSession = config.authProvider === "simple" || config.authProvider === "oidc";

// The bell is the only place that watches for a fire the reader has not seen, so
// it owns the polling. Without a cloud link there is nothing to remember and
// nothing to poll, and the page it opens lands on the rules like it always did.
const { unseen: unseenAlerts, fetchRecentAlerts, refreshRecentAlerts } = useRecentAlerts();

onMounted(() => fetchRecentAlerts());

// A minute is slow enough to be invisible and fast enough that the dot is not a
// lie. Paused while the tab is hidden, and caught up the moment it comes back.
const visibility = useDocumentVisibility();
const { pause, resume } = useIntervalFn(() => refreshRecentAlerts(), 60_000);
watch(visibility, (state) => {
  if (state === "visible") {
    refreshRecentAlerts();
    resume();
  } else {
    pause();
  }
});

type LogoutTarget = { url: string; params?: Record<string, string> };

async function logout() {
  let target: LogoutTarget | undefined = logoutUrl ? { url: logoutUrl } : undefined;

  if (hasSession) {
    const response = await fetch(withBase("/api/token"), {
      method: "DELETE",
    });
    // Under oidc the server answers with the issuer's logout, since only it can
    // read the ID token the session was issued with.
    if (response.headers.get("Content-Type")?.includes("application/json")) {
      target = await response.json();
    }
  }

  if (!target) {
    location.reload();
  } else if (target.params) {
    // A form POST, because the ID token in the hint can be several KB, past the
    // request line a proxy in front of the issuer accepts on a GET.
    const form = document.createElement("form");
    form.method = "POST";
    form.action = target.url;
    for (const [name, value] of Object.entries(target.params)) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = name;
      input.value = value;
      form.appendChild(input);
    }
    document.body.appendChild(form);
    form.submit();
  } else {
    location.href = target.url;
  }
}
</script>
