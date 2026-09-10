<template>
  <button class="btn relative overflow-hidden" @click="cancel()">
    <div class="absolute inset-0 origin-left bg-white/30" ref="progress"></div>
    <div>
      <slot></slot>
    </div>
  </button>
</template>

<script lang="ts" setup>
const progress = ref<HTMLElement>();
const finished = defineEmit();
const cancelled = defineEmit();
let animation: Animation | undefined;

const { duration = 4000 } = defineProps<{
  duration?: number;
}>();

onMounted(async () => {
  animation = progress.value?.animate([{ transform: "scaleX(0)" }, { transform: "scaleX(1)" }], {
    duration: duration,
    easing: "linear",
    fill: "forwards",
  });
  try {
    await animation?.finished;
    finished();
  } catch (e) {
    progress.value?.animate([{ transform: "scaleX(1)" }, { transform: "scaleX(0)" }], {
      duration: 0,
      fill: "forwards",
    });
    cancelled();
  }
});

const cancel = () => {
  animation?.cancel();
};
</script>

<style scoped></style>
