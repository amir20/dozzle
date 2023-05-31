<template>
  <dropdown-menu class="is-right">
    <a class="dropdown-item" @click="clear()">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon:trash-24 class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.clear") }}</div>
        </div>
      </div>
    </a>
    <a class="dropdown-item" :href="`${base}/api/logs/download?id=${container.id}&host=${sessionHost}`">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon:download-24 class="mr-4" />
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
            <mdi:light-magnify class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.search") }}</div>
        </div>
      </div>
    </a>

    <hr class="dropdown-divider" />
    <a class="dropdown-item" @click="streamConfig.stdout = !streamConfig.stdout">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <mdi:check class="mr-4 is-blue" v-if="streamConfig.stdout" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.stdout") }}</div>
        </div>
      </div>
    </a>
    <a class="dropdown-item" @click="streamConfig.stderr = !streamConfig.stderr">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <mdi:check class="mr-4 is-red" v-if="streamConfig.stderr" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.stderr") }}</div>
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
  width: 3em;
}
</style>
