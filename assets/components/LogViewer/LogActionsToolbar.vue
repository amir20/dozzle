<template>
  <dropdown-menu class="is-right">
    <template #trigger>
      <button class="button" aria-haspopup="true" aria-controls="dropdown-menu">
        <span class="icon">
          <carbon:circle-solid class="is-red is-small" v-if="streamConfig.stderr" />
          <carbon:circle-solid class="is-blue is-small" v-if="streamConfig.stdout" />
        </span>
      </button>
    </template>
    <a class="dropdown-item" @click="clear()">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon:trash-24 />
          </div>
        </div>
        <div class="level-right is-justify-content-space-between is-flex-grow-1">
          <div class="level-item">{{ $t("toolbar.clear") }}</div>
          <div class="level-item"><key-shortcut char="k" :modifiers="['shift', 'meta']"></key-shortcut></div>
        </div>
      </div>
    </a>
    <a class="dropdown-item" :href="`${base}/api/logs/download/${container.host}/${container.id}`">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon:download-24 />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.download") }}</div>
        </div>
      </div>
    </a>
    <hr class="dropdown-divider" />
    <a class="dropdown-item" @click="showSearch = true">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <mdi:light-magnify />
          </div>
        </div>
        <div class="level-right is-justify-content-space-between is-flex-grow-1">
          <div class="level-item">{{ $t("toolbar.search") }}</div>
          <div class="level-item"><key-shortcut char="f"></key-shortcut></div>
        </div>
      </div>
    </a>

    <hr class="dropdown-divider" />
    <a
      class="dropdown-item"
      @click="
        streamConfig.stdout = true;
        streamConfig.stderr = true;
      "
    >
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <template v-if="streamConfig.stderr && streamConfig.stdout">
              <carbon:circle-solid class="is-red is-small" />
              <carbon:circle-solid class="is-blue is-small" />
            </template>
          </div>
        </div>
        <div class="level-right">
          {{ $t("toolbar.show-all") }}
        </div>
      </div>
    </a>
    <a
      class="dropdown-item"
      @click="
        streamConfig.stdout = true;
        streamConfig.stderr = false;
      "
    >
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <carbon:circle-solid class="is-blue is-small" v-if="!streamConfig.stderr && streamConfig.stdout" />
          </div>
        </div>
        <div class="level-right">
          {{ $t("toolbar.show", { std: "STDOUT" }) }}
        </div>
      </div>
    </a>
    <a
      class="dropdown-item"
      @click="
        streamConfig.stdout = false;
        streamConfig.stderr = true;
      "
    >
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <carbon:circle-solid class="is-red is-small" v-if="streamConfig.stderr && !streamConfig.stdout" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">
            {{ $t("toolbar.show", { std: "STDERR" }) }}
          </div>
        </div>
      </div>
    </a>
  </dropdown-menu>
</template>

<script lang="ts" setup>
import { type ComputedRef } from "vue";
import { Container } from "@/models/Container";

const { showSearch } = useSearchFilter();
const { base } = config;

const clear = defineEmit();

const container = inject("container") as ComputedRef<Container>;
const streamConfig = inject("stream-config") as { stdout: boolean; stderr: boolean };
</script>

<style lang="scss" scoped>
.level-left .level-item {
  width: 2.2em;
  align-items: center;
  margin-right: 0.5em;
}

.is-small {
  width: 0.6em;
}
</style>
