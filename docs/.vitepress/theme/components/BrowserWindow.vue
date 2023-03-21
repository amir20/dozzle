<script setup></script>

<template>
  <div border="~ solid rounded light-100 dark:dark-50" bg="[#eee] dark:[#222]">
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
