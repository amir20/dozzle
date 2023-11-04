<template>
  <article class="prose p-8" v-html="data.content"></article>
</template>

<script lang="ts" setup>
const { id } = defineProps<{ id: string }>();
const data = ref({ title: "", content: "" });

onBeforeMount(async () => {
  data.value = await (await fetch(withBase("/api/content/" + id))).json();

  setTitle(data.value.title);
});
</script>
<style lang="postcss" scoped></style>
