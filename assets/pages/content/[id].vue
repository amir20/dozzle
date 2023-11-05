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
const data = ref({ title: "", content: "" });

onBeforeMount(async () => {
  data.value = await (await fetch(withBase("/api/content/" + id))).json();

  setTitle(data.value.title);
});
</script>
<style lang="postcss" scoped></style>
