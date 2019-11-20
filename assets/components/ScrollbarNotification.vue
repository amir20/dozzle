<template lang="html">
  <transition name="fade">
    <div>
      <button class="button" :class="hasNew ? 'is-warning' : 'is-primary'" @click="scrollToBottom" v-show="visible">
        <span class="icon large"> <i class="fas fa-chevron-down"></i> </span>
      </button>
    </div>
  </transition>
</template>

<script>
let scroballeView;
export default {
  props: ["messages"],
  data() {
    return {
      visible: false,
      hasNew: false,
      scroballeView: null
    };
  },
  mounted() {
    this.scroballeView = this.$el.closest(".is-scrollable");
    this.scroballeView.addEventListener("scroll", this.onScroll, { passive: true });
    setTimeout(() => this.scrollToBottom(), 500);
  },
  beforeDestroy() {
    this.scroballeView.removeEventListener("scroll", this.onScroll);
  },
  methods: {
    scrollToBottom() {
      this.visible = false;
      this.scroballeView.scrollTo(0, this.scroballeView.scrollHeight);
    },
    onScroll() {
      const scrollTop = this.scroballeView.scrollTop;
      const scrollBottom = this.scroballeView.scrollHeight - this.scroballeView.clientHeight;
      this.visible = scrollBottom - scrollTop > 50;
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
div {
  text-align: right;
  margin-right: 65px;
}

button {
  position: fixed;
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
