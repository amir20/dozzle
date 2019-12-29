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
        <b-switch v-model="search">
          Enable searching with Dozzle using <code>command+f</code> or <code>ctrl+f</code>
        </b-switch>
      </div>

      <div class="item">
        <h2 class="title is-6 is-marginless">Font size</h2>

        Modify the font size when viewing logs.
        <br /><br />
        <b-dropdown v-model="size" aria-role="list">
          <button class="button is-primary" type="button" slot="trigger">
            <span class="is-capitalized">{{ size }}</span>
            <span class="icon"><ion-icon name="ios-arrow-down"></ion-icon></span>
          </button>
          <b-dropdown-item :value="value" aria-role="listitem" v-for="value in ['small', 'medium', 'large']">
            <div class="media">
              <span class="icon">
                <ion-icon name="checkmark" v-if="value == size"></ion-icon>
              </span>
              <div class="media-content">
                <h3 class="is-capitalized">{{ value }}</h3>
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
import { mapActions, mapState } from "vuex";

function computedSettings(names) {
  return names.reduce((map, name) => {
    map[name] = {
      get() {
        return this.settings[name];
      },
      set(value) {
        this.updateSetting({ [name]: value });
      }
    };
    return map;
  }, {});
}

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
  methods: {
    ...mapActions({
      updateSetting: "UPDATE_SETTING"
    })
  },
  computed: {
    ...mapState(["settings"]),
    ...computedSettings.bind(this)(["search", "size"])
  }
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
  padding: 1rem 1.5rem;
}

.has-underline {
  border-bottom: 1px solid #fff;
  padding: 1em 0px;
  margin-bottom: 1em;
}

.item {
  padding: 1em 0;
}

code {
  border-radius: 4px;
  background-color: #444;
}
</style>
