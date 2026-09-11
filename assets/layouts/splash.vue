<template>
  <div class="hero max-h-500:pt-4 min-h-dvh content-start pt-[20dvh]">
    <!-- Full width so the card inside can cap itself against the viewport. Left to
         shrink-to-fit, a fixed-width card sets the hero's width and pushes the page
         sideways on a phone instead of being narrowed by it. -->
    <div class="hero-content w-full">
      <router-view></router-view>
    </div>
  </div>
</template>
