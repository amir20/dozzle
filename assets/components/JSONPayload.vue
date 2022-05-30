<template>
  <ul class="fields">
    <li v-for="(value, name) in data">
      <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
    </li>
  </ul>
</template>
<script lang="ts" setup>
import { computed, PropType, ref } from "vue";

const props = defineProps({
  payload: {
    type: Object as PropType<Record<string, any>>,
    required: true,
  },
});

const attributes = ref([["msg"], ["request", "uri"]]);

function getDeep(obj: Record<string, any>, path: string[]) {
  console.log(obj, path);
  return path.reduce((acc, key) => acc?.[key], obj);
}
const data = computed(() =>
  attributes.value.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(props.payload, attr) }), {})
);
</script>

<style lang="scss" scoped>
.fields {
  display: inline-block;
  list-style: none;

  li {
    display: inline-block;
    margin-left: 1em;
  }
}
</style>
