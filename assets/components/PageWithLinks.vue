<template>
  <div class="flex flex-col gap-5 px-4 py-4 md:px-8">
    <section class="flex items-center gap-4">
      <CloudSearchInline class="hidden max-w-sm flex-1 md:flex" />
      <Links class="ml-auto">
        <template #more-items>
          <Tag class="font-mono">{{ config.version }}</Tag>
        </template>
      </Links>
    </section>
    <slot></slot>
  </div>
</template>
