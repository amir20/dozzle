<template>
  <Dropdown :placement="placement" :trigger-class="triggerClass" data-testid="user-menu" v-if="config.user">
    <template #trigger>
      <template v-if="config.disableAvatars || !config.user.email">
        <material-symbols:person-outline v-if="outline" :class="iconClass" />
        <material-symbols:person v-else :class="iconClass" />
      </template>
      <template v-else>
        <img
          class="ring-base-content/25 max-w-none rounded-full p-px ring-1"
          :class="iconClass"
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
</template>

<script lang="ts" setup>
const {
  placement = "bottom-end",
  iconClass = "size-6",
  triggerClass,
} = defineProps<{
  placement?: PopoverPlacement;
  iconClass?: string;
  triggerClass?: string;
  outline?: boolean;
}>();

const { logoutUrl } = config;

// simple and oidc hold a session cookie Dozzle issued, so logging out means
// clearing it. Forward proxy has none: the logout URL is the whole logout there.
const hasSession = config.authProvider === "simple" || config.authProvider === "oidc";

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
