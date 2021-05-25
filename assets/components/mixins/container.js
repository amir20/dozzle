import { mapGetters } from "vuex";
export default {
  computed: {
    ...mapGetters(["allContainersById"]),
    container() {
      return this.allContainersById[this.id];
    },
  },
  watch: {
    ["container.state"](newValue, oldValue) {
      if (newValue == "running" && newValue != oldValue) {
        this.onContainerStateChange(newValue, oldValue);
      }
    },
  },
  methods: {
    onContainerStateChange(newValue, oldValue) {},
  },
};
