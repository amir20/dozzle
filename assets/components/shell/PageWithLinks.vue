<template>
  <div class="flex flex-col gap-5 p-4 md:px-8">
    <!-- On a phone the bar is fixed and opaque. Fixed rather than sticky because the
         splitpanes it sits in clip their overflow, which pins a sticky child to the
         pane instead of the screen. Opaque because iOS 26+ blurs whatever is drawn
         just under the status bar; a solid bar at the top edge gets a solid status
         area to match instead of a blur over the logo. -->
    <section
      class="max-md:bg-base-200 max-md:border-base-content/10 flex items-center gap-4 max-md:fixed max-md:inset-x-0 max-md:top-0 max-md:z-20 max-md:h-[calc(4rem+env(safe-area-inset-top))] max-md:border-b max-md:px-4 max-md:pt-[env(safe-area-inset-top)]"
    >
      <!-- A phone has no sidebar or inline search, so without the wordmark the left
           half of the bar is empty. Same mark the Browse tab leads with. -->
      <div class="flex min-w-0 items-center gap-2 md:hidden">
        <BrandMark small />
      </div>
      <SearchInline class="hidden max-w-sm flex-1 md:flex" />
      <Links class="ml-auto">
        <template #more-items>
          <!-- Settings › About has it on a phone, where the top bar has no room to spare. -->
          <Tag class="font-mono max-md:hidden">{{ config.version }}</Tag>
        </template>
      </Links>
    </section>
    <!-- Holds the fixed bar's place in the flow, less the page padding above it. -->
    <div class="h-[calc(3rem+env(safe-area-inset-top))] shrink-0 md:hidden" aria-hidden="true"></div>
    <slot></slot>
  </div>
</template>
<script lang="ts" setup></script>
