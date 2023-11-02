<template>
  <article class="prose p-8 lg:prose-xl" v-html="content"></article>
</template>

<script lang="ts" setup>
const content = ref("");

onBeforeMount(async () => {
  content.value = await (await fetch(withBase("/api/content"))).text();
});
</script>
<style lang="postcss" scoped></style>
