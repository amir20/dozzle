<template lang="html">
  <transition name="fade">
    <button
      class="button scroll-notification"
      :class="hasNew ? 'is-warning' : 'is-primary'"
      @click="scrollToBottom"
      v-show="visible"
    >
      <span class="icon large"> <i class="fas fa-chevron-down"></i> </span>
    </button>
  </transition>
</template>

<script>
export default {
  props: ["messages"],
  data() {
    return {
      visible: false,
      hasNew: false
    };
  },
  mounted() {
    document.addEventListener("scroll", this.onScroll, { passive: true });
    setTimeout(() => this.scrollToBottom(), 500);
  },
  beforeDestroy() {
    document.removeEventListener("scroll", this.onScroll);
  },
  methods: {
    scrollToBottom() {
      this.visible = false;
      window.scrollTo(0, document.documentElement.scrollHeight || document.body.scrollHeight);
    },
    onScroll() {
      const scrollTop = document.documentElement.scrollTop || document.body.scrollTop;
      const scrollBottom =
        (document.documentElement.scrollHeight || document.body.scrollHeight) - document.documentElement.clientHeight;
      const diff = Math.abs(scrollTop - scrollBottom);
      this.visible = diff > 50;
      if (!this.visible) {
        this.hasNew = false;
      }
    }
  },
  watch: {
    messages() {
      if (this.visible) {
        this.hasNew = true;
      } else {
        this.scrollToBottom();
      }
    }
  }
};
</script>
<style scoped>
.scroll-notification {
  position: fixed;
  right: 40px;
  bottom: 30px;
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease-in;
}
.fade-enter,
.fade-leave-to {
  opacity: 0;
}
</style>
