<template>
  <div class="flex flex-col gap-5 p-4 md:px-8">
    <section class="flex items-center gap-4">
      <SearchInline class="hidden max-w-sm flex-1 md:flex" />
      <Links class="ml-auto">
        <template #more-items>
          <!-- Settings › About has it on a phone, where the top bar has no room to spare. -->
          <Tag class="font-mono max-md:hidden">{{ config.version }}</Tag>
        </template>
      </Links>
    </section>
    <slot></slot>
  </div>
</template>
