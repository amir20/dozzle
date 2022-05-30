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
    type: Object as PropType<object>,
    required: true,
  },
});

const attributes = ref(["msg"]);

const data = computed(() => attributes.value.reduce((acc, attr) => ({ ...acc, [attr]: props.payload[attr] }), {}));
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
