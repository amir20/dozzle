<script lang="ts" setup>
// Real quotes only. Drop in verbatim text with a link back to the source
// (GitHub issue, HN thread, r/selfhosted post) so anyone can check it.
// The section hides itself while this is empty.
type Testimonial = { quote: string; author: string; source: string; href: string };

const testimonials: Testimonial[] = [];
</script>

<template>
  <section v-if="testimonials.length" class="mx-auto mt-20 max-w-[1152px] px-6">
    <h2 class="m-0! text-center text-2xl font-semibold text-(--vp-c-text-1) md:text-3xl">What people say</h2>

    <div class="mt-10 grid gap-4 md:grid-cols-3">
      <figure
        v-for="testimonial in testimonials"
        :key="testimonial.href"
        class="m-0! flex flex-col rounded-xl border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-alt) p-6"
      >
        <Icon icon="mdi:format-quote-close" class="size-6 text-(--vp-c-text-3) opacity-60" />
        <blockquote class="mt-3! mb-0! border-0! p-0! text-(--vp-c-text-1)">{{ testimonial.quote }}</blockquote>
        <figcaption class="mt-4! text-sm text-(--vp-c-text-2)">
          {{ testimonial.author }} on
          <a :href="testimonial.href">{{ testimonial.source }}</a>
        </figcaption>
      </figure>
    </div>
  </section>
</template>
