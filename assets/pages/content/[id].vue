<template>
  <div class="flex flex-col gap-8 p-8">
    <section>
      <links />
    </section>
    <section>
      <article class="prose" v-html="data.content"></article>
    </section>
  </div>
</template>

<script lang="ts" setup>
const { id } = defineProps<{ id: string }>();

const { data } = await useFetch(() => withBase("/api/content/" + id), {
  refetch: true,
})
  .get()
  .json<{ title: string; content: string }>();
</script>
<style lang="postcss" scoped></style>
