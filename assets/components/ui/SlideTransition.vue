<template>
  <transition :name="!slideRight ? 'slide-left' : 'slide-right'" mode="out-in">
    <div v-if="!slideRight">
      <slot name="left" />
    </div>
    <div v-else>
      <slot name="right" />
    </div>
  </transition>
</template>

<script lang="ts" setup>
const { slideRight } = defineProps<{ slideRight: boolean }>();
</script>
<style scoped>
.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all 0.1s ease-out;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-right-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.slide-right-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>
