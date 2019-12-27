<template lang="html">
  <div class="is-fullheight">
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">About</h2>
      </div>

      <h2 class="title is-6 is-marginless">Version</h2>
      <div>
        You are using Dozzle <i>{{ currentVersion }}</i
        >.
        <span v-if="hasUpdate">
          New version is available! Update to
          <a :href="nextRelease.html_url" class="next-release">{{ nextRelease.name }}</a
          >.
        </span>
      </div>
    </section>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">Display</h2>
      </div>
      <div class="item">
        <b-switch>Switch rounded default</b-switch>
      </div>

      <div class="item">
        <b-dropdown v-model="isPublic" aria-role="list">
          <button class="button is-primary" type="button" slot="trigger">
            <template v-if="isPublic">
              <span>Public</span>
            </template>
            <template v-else>
              <span>Friends</span>
            </template>
            <span class="icon"><ion-icon name="ios-arrow-down"></ion-icon></span>
          </button>

          <b-dropdown-item :value="true" aria-role="listitem">
            <div class="media">
              <b-icon class="media-left" icon="earth"></b-icon>
              <div class="media-content">
                <h3>Public</h3>
                <small>Everyone can see</small>
              </div>
            </div>
          </b-dropdown-item>

          <b-dropdown-item :value="false" aria-role="listitem">
            <div class="media">
              <b-icon class="media-left" icon="account-multiple"></b-icon>
              <div class="media-content">
                <h3>Friends</h3>
                <small>Only friends can see</small>
              </div>
            </div>
          </b-dropdown-item>
        </b-dropdown>
      </div>
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
<style lang="scss">
.is-fullheight {
  min-height: 100vh;
}

.title {
  color: #eee;
}

a.next-release {
  text-decoration: underline;
  color: #00d1b2;

  &:hover {
    text-decoration: none;
  }
}

.section {
  padding: 2rem 1.5rem;
}

.has-underline {
  border-bottom: 1px solid #fff;
  padding: 1em 0px;
  margin-bottom: 1em;
}

.item {
  padding: 1em 0;
}
</style>
