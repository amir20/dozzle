<template lang="html">
  <div class="is-fullheight">
    <section class="section">
      <h2 class="title is-6 is-marginless">Version</h2>
      <div>
        You are using Dozzle <i>{{ currentVersion }}</i
        >.
        <span v-if="hasUpdate">
          New version is available! Update to
          <a :href="nextRelease.html_url">{{ nextRelease.name }}</a
          >.
        </span>
      </div>
    </section>
    <section class="section">
      <h2 class="title is-6 is-marginless">Remember Menu Width</h2>
      <input id="switchRoundedDefault" type="checkbox" class="switch is-rounded is-rtl" checked="checked" />
      <label for="switchRoundedDefault">Switch rounded default</label>
    </section>
  </div>
</template>

<script>
import gt from "semver/functions/gt";
import valid from "semver/functions/valid";

export default {
  props: [],
  name: "Settings",
  components: {},
  data() {
    return {
      currentVersion: VERSION,
      nextRelease: null,
      hasUpdate: false
    };
  },
  async created() {
    const releases = await (await fetch("https://api.github.com/repos/amir20/dozzle/releases")).json();
    this.hasUpdate = gt(releases[0].tag_name, this.currentVersion);
    this.nextRelease = releases[0];
  },
  metaInfo() {
    return {
      title: "Settings",
      titleTemplate: "%s - Dozzle"
    };
  },

  watch: {},
  methods: {},
  computed: {},
  filters: {}
};
</script>
<style scoped lang="scss">
@import "~bulma-switch";

.is-fullheight {
  min-height: 100vh;
}

.title {
  color: #eee;
}

a {
  text-decoration: underline;
  color: #00d1b2;

  &:hover {
    text-decoration: none;
  }
}

.section {
  padding: 2rem 1.5rem;
}
</style>
