<template>
  <dropdown-menu class="is-right">
    <a class="dropdown-item" @click="onClearClicked">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon-trash-24 class="mr-4" />
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
            <octicon-download-24 class="mr-4" />
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
            <mdi-light-magnify class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">{{ $t("toolbar.search") }}</div>
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

const { onClearClicked = (e: Event) => {} } = defineProps<{
  onClearClicked: (e: Event) => void;
}>();

const container = inject("container") as ComputedRef<Container>;
</script>

<style lang="scss" scoped>
#download.button,
#clear.button {
  .icon {
    height: 80%;
  }

  &:hover {
    color: var(--primary-color);
    border-color: var(--primary-color);
  }
}
</style>
