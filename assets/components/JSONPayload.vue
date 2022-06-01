<template>
  <ul class="fields" @click="expanded = !expanded">
    <li v-for="(value, name) in data">
      <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
    </li>
  </ul>
  <field-list :fields="payload" :expanded="expanded"></field-list>
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
  return path.reduce((acc, key) => acc?.[key], obj);
}
const data = computed(() =>
  attributes.value.reduce((acc, attr) => ({ ...acc, [attr.join(".")]: getDeep(props.payload, attr) }), {})
);

const expanded = ref(false);
</script>

<style lang="scss" scoped>
.fields {
  display: inline-block;
  list-style: none;

  &:hover {
    cursor: pointer;
  }

  li {
    display: inline-block;
    margin-left: 1em;
  }
}
</style>
