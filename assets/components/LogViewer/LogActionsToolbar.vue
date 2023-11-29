<template>
  <div class="dropdown dropdown-end dropdown-hover">
    <label tabindex="0" class="btn btn-ghost btn-sm gap-0.5 px-2">
      <carbon:circle-solid class="w-2.5 text-red" v-if="streamConfig.stderr" />
      <carbon:circle-solid class="w-2.5 text-blue" v-if="streamConfig.stdout" />
    </label>
    <ul tabindex="0" class="menu dropdown-content z-50 w-52 rounded-box bg-base p-1 shadow">
      <li>
        <a @click.prevent="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <key-shortcut char="k" :modifiers="['shift', 'meta']"></key-shortcut>
        </a>
      </li>
      <li>
        <a :href="`${base}/api/logs/download/${container.host}/${container.id}`" download>
          <octicon:download-24 /> {{ $t("toolbar.download") }}
        </a>
      </li>
      <li>
        <a @click.prevent="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <key-shortcut char="f"></key-shortcut>
        </a>
      </li>
      <li class="line"></li>
      <li>
        <a
          @click="
            streamConfig.stdout = true;
            streamConfig.stderr = true;
          "
        >
          <div class="flex h-4 w-4 gap-0.5">
            <template v-if="streamConfig.stderr && streamConfig.stdout">
              <carbon:circle-solid class="w-2 text-red" />
              <carbon:circle-solid class="w-2 text-blue" />
            </template>
          </div>
          {{ $t("toolbar.show-all") }}
        </a>
      </li>
      <li>
        <a
          @click="
            streamConfig.stdout = true;
            streamConfig.stderr = false;
          "
        >
          <div class="flex h-4 w-4 flex-col gap-1">
            <carbon:circle-solid class="w-2 text-blue" v-if="!streamConfig.stderr && streamConfig.stdout" />
          </div>
          {{ $t("toolbar.show", { std: "STDOUT" }) }}
        </a>
      </li>
      <li>
        <a
          @click="
            streamConfig.stdout = false;
            streamConfig.stderr = true;
          "
        >
          <div class="flex h-4 w-4 flex-col gap-1">
            <carbon:circle-solid class="w-2 text-red" v-if="streamConfig.stderr && !streamConfig.stdout" />
          </div>
          {{ $t("toolbar.show", { std: "STDERR" }) }}
        </a>
      </li>
      <li class="line"></li>
      <ul v-show="enableActions">
        <li>
          <button
            @click="() => actionHandler('stop')"
            :disabled="actionStates['stop-action']"
            v-if="container.state == 'running'"
          >
            <carbon:stop-filled-alt /> Stop
          </button>
          <button
            @click="() => actionHandler('start')"
            :disabled="actionStates['start-action']"
            v-if="container.state != 'running'"
          >
            <carbon:play /> Start
          </button>
        </li>
        <li>
          <button @click="() => actionHandler('restart')" :disabled="actionStates['restart-action']">
            <carbon:restart
              :class="{
                'animate-spin': actionStates['restart-action'],
                'text-secondary': actionStates['restart-action'],
              }"
            />
            Restart
          </button>
        </li>
      </ul>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { ContainerActions } from "@/types/Container";

const { showSearch } = useSearchFilter();
const { base, enableActions } = config;
const { showToast } = useToast();

const clear = defineEmit();

const { container, streamConfig } = useContainerContext();

const actionStates = reactive({
  "stop-action": false,
  "restart-action": false,
  "start-action": false,
});

async function actionHandler(action: ContainerActions) {
  const actionUrl = `/api/actions/${action}/${container.value.id}`;
  const errorToast = (message?: string) =>
    showToast(
      {
        type: "error",
        message: message ?? "Something went wrong",
        title: "Action failed",
      },
      { expire: 5000 },
    );

  actionStates[`${action}-action`] = true;

  await fetch(withBase(actionUrl))
    .then((res) => {
      if (res.status === 404) {
        errorToast("Container not found");
      } else if (res.status === 500) {
        errorToast();
      }
    })
    .catch((e) => errorToast(e.message));

  actionStates[`${action}-action`] = false;
}
</script>

<style scoped lang="postcss">
li.line {
  @apply h-px bg-base-content/20;
}

a {
  @apply whitespace-nowrap;
}
</style>
