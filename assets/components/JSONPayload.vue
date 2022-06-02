<template>
  <ul class="fields" @click="expanded = !expanded">
    <li v-for="(value, name) in data">
      <template v-if="value">
        <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
      </template>
    </li>
  </ul>
  <field-list :fields="payload" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
</template>
<script lang="ts" setup>
import { flattenJSON, getDeep } from "@/utils";
import { computed, PropType, ref } from "vue";

const props = defineProps({
  payload: {
    type: Object as PropType<Record<string, any>>,
    required: true,
  },
  visibleKeys: {
    type: Array as PropType<string[][]>,
    default: [],
  },
});

const expanded = ref(false);

const data = computed(() => {
  if (!props.visibleKeys.length) {
    return flattenJSON(props.payload);
  } else {
    return props.visibleKeys.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(props.payload, attr) }), {});
  }
});
</script>

<style lang="scss" scoped>
.fields {
  display: inline-block;
  list-style: none;

  &:hover {
    cursor: pointer;
    &::after {
      content: "expand json";
      color: var(--secondary-color);
      display: inline-block;
      margin-left: 0.5em;
      font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
    }
  }

  li {
    display: inline-block;
    margin-left: 1em;
  }
}
</style>
