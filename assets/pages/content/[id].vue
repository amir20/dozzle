<template>
  <page-with-links>
    <section>
      <article class="prose" v-html="data.content" v-if="data"></article>
    </section>
  </page-with-links>
</template>

<script lang="ts" setup>
const { id } = defineProps<{ id: string }>();

const { data } = useFetch(() => withBase("/api/content/" + id), {
  refetch: true,
})
  .get()
  .json<{ title: string; content: string }>();
</script>
<style lang="postcss" scoped></style>
