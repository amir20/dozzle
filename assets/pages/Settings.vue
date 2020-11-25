<template>
  <div>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">About</h2>
      </div>

      <div>
        You are using Dozzle <i>{{ currentVersion }}</i
      >.
        <span v-if="hasUpdate">
          New version is available! Update to
          <a :href="nextRelease.html_url" class="next-release" target="_blank" rel="noreferrer noopener">{{
              nextRelease.name
            }}</a
          >.
        </span>
      </div>
    </section>

    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">Display</h2>
      </div>
      <div class="item">
        By default, Dozzle will use your browser's locale to format time. You can force to 12 or 24 hour style.
        <br />
        <br />
        <b-field>
          <b-radio-button
            v-model="hourStyle"
            :native-value="value"
            v-for="value in ['auto', '12', '24']"
            :key="value"
          >
            <span class="is-capitalized">{{ value }}</span>
          </b-radio-button>
        </b-field>
        <div class="item">
          <b-switch v-model="smallerScrollbars"> Use smaller scrollbars </b-switch>
        </div>
        <div class="item">
          <b-switch v-model="showTimestamp"> Show timestamps </b-switch>
        </div>
      </div>

      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <b-field>
              <b-radio-button
                v-model="size"
                :native-value="value"
                v-for="value in ['small', 'medium', 'large']"
                :key="value"
              >
                <span class="is-capitalized">{{ value }}</span>
              </b-radio-button>
            </b-field>
          </div>
          <div class="column">Font size to use for logs</div>
        </div>
      </div>
    </section>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">Options</h2>
      </div>

      <div class="item">
        <b-switch v-model="search">
          Enable searching with Dozzle using <code>command+f</code> or <code>ctrl+f</code>
        </b-switch>
      </div>

      <div class="item">
        <b-switch v-model="showAllContainers"> Show stopped containers </b-switch>
      </div>

      <div class="item">
        <b-switch v-model="lightTheme"> Use light theme </b-switch>
      </div>

      <div class="item">
        <b-switch v-model="checkingUpdates"> Checking updates </b-switch>
      </div>
    </section>
  </div>
</template>

<script>
import gt from "semver/functions/gt";
import valid from "semver/functions/valid";
import { mapActions, mapState } from "vuex";
import Icon from "../components/Icon";
import config from "../store/config";

export default {
  props: [],
  name: "Settings",
  components: {
    Icon,
  },
  data() {
    return {
      currentVersion: config.version,
      nextRelease: null,
      hasUpdate: false,
    };
  },
  async created() {
    if (this.settings.checkingUpdates) {
      const releases = await (await fetch("https://api.github.com/repos/amir20/dozzle/releases")).json();
      if (this.currentVersion !== "dev") {
        this.hasUpdate = gt(releases[0].tag_name, this.currentVersion);
      } else {
        this.hasUpdate = true;
      }
      this.nextRelease = releases[0];
    }
  },
  metaInfo() {
    return {
      title: "Settings",
    };
  },
  methods: {
    ...mapActions({
      updateSetting: "UPDATE_SETTING",
    }),
  },
  computed: {
    ...mapState(["settings"]),
    ...["search", "size", "smallerScrollbars", "showTimestamp", "showAllContainers", "lightTheme", "hourStyle", 'checkingUpdates'].reduce(
      (map, name) => {
        map[name] = {
          get() {
            return this.settings[name];
          },
          set(value) {
            this.updateSetting({ [name]: value });
          },
        };
        return map;
      },
      {}
    ),
  },
};
</script>
<style lang="scss" scoped>
.title {
  color: var(--title-color);
}

a.next-release {
  text-decoration: underline;
  &:hover {
    text-decoration: none;
  }
}

.section {
  padding: 1rem 1.5rem;
}

.has-underline {
  border-bottom: 1px solid var(--border-color);
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
