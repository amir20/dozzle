<template>
  <section :class="{ 'is-full-height-scrollable': scrollable }">
    <header v-if="$slots.header">
      <slot name="header"></slot>
    </header>
    <main :data-scrolling="scrollable ? true : undefined">
      <div class="is-scrollbar-progress is-hidden-mobile">
        <scroll-progress v-show="paused" :indeterminate="loading" :auto-hide="!loading"></scroll-progress>
      </div>
      <div ref="scrollableContent">
        <slot :setLoading="setLoading"></slot>
      </div>

      <div ref="scrollObserver" class="is-scroll-observer"></div>
    </main>

    <div class="is-scrollbar-notification">
      <transition name="fade">
        <button
          class="button has-boxshadow"
          :class="hasMore ? 'has-more' : ''"
          @click="scrollToBottom()"
          v-show="paused"
        >
          <mdi:light-chevron-double-down />
        </button>
      </transition>
    </div>
  </section>
</template>

<script lang="ts" setup>
const { scrollable = false } = defineProps<{ scrollable?: boolean }>();

let paused = $ref(false);
let hasMore = $ref(false);
let loading = $ref(false);
const scrollObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();

provide("scrollingPaused", $$(paused));

const mutationObserver = new MutationObserver((e) => {
  if (!paused) {
    scrollToBottom();
  } else {
    const record = e[e.length - 1];
    if (record.target.children[record.target.children.length - 1] == record.addedNodes[record.addedNodes.length - 1]) {
      hasMore = true;
    }
  }
});

const intersectionObserver = new IntersectionObserver((entries) => (paused = entries[0].intersectionRatio == 0), {
  threshold: [0, 1],
  rootMargin: "80px 0px",
});

onMounted(() => {
  mutationObserver.observe(scrollableContent.value!, { childList: true, subtree: true });
  intersectionObserver.observe(scrollObserver.value!);
});

function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore = false;
}

function setLoading(value: boolean) {
  loading = value;
}
</script>
<style scoped lang="scss">
section {
  display: flex;
  flex-direction: column;

  header {
    position: sticky;
    top: 0;
    background: var(--body-background-color);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    z-index: 1;
  }

  &.is-full-height-scrollable {
    height: 100vh;
    min-height: 0;
  }

  main {
    flex: 1;
    overflow: auto;
    scroll-snap-type: y proximity;
  }

  .is-scrollbar-progress {
    text-align: right;
    margin-right: 110px;

    .scroll-progress {
      position: fixed;
      top: 60px;
      z-index: 2;
    }
  }

  .is-scroll-observer {
    height: 1px;
  }

  .is-scrollbar-notification {
    text-align: right;
    margin-right: 65px;

    button {
      position: fixed;
      bottom: 30px;
      background-color: var(--primary-color);
      transition: background-color 0.24s ease-out;
      border: none !important;
      color: #eee;

      &.has-more {
        background-color: var(--secondary-color);
        animation-name: bounce;
        animation-duration: 1000ms;
        animation-fill-mode: both;

        color: #222;
      }
    }
  }

  @keyframes bounce {
    0%,
    20%,
    50%,
    80%,
    100% {
      transform: translateY(0);
    }

    40% {
      transform: translateY(-30px);
    }

    60% {
      transform: translateY(-15px);
    }
  }

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.15s ease-in;
  }

  .fade-enter,
  .fade-leave-to {
    opacity: 0;
  }
}
</style>
