<script setup></script>

<template>
  <div border="~ solid rounded dark-50" bg="[#121212]">
    <div flex gap="2" p-2>
      <div circle-red />
      <div circle-yellow />
      <div circle-green />
    </div>
    <div class="content">
      <slot />
    </div>
  </div>
</template>
