<template>
  <div class="flex flex-col gap-5 px-4 py-4 md:px-8">
    <section>
      <Links>
        <template #more-items>
          <Tag>{{ config.version }}</Tag>
        </template>
      </Links>
    </section>
    <slot></slot>
  </div>
</template>
