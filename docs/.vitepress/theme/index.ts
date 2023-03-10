// https://vitepress.dev/guide/custom-theme
import { h } from "vue";
import Theme from "vitepress/theme";
import "uno.css";
import "./style.css";
import HeroVideo from "./components/HeroVideo.vue";

export default {
  ...Theme,
  Layout: () => {
    return h(Theme.Layout, null, {
      "home-hero-image": () => h(HeroVideo),
    });
  },
  enhanceApp({ app, router, siteData }) {},
};
