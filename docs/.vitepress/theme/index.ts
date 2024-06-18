// https://vitepress.dev/guide/custom-theme
import { h } from "vue";
import Theme from "vitepress/theme";
import "./style.css";
import HeroVideo from "./components/HeroVideo.vue";
import BuyMeCoffee from "./components/BuyMeCoffee.vue";
import Stats from "./components/Stats.vue";

export default {
  ...Theme,
  Layout: () => {
    return h(Theme.Layout, null, {
      "home-hero-image": () => h(HeroVideo),
      "sidebar-nav-after": () => h(BuyMeCoffee),
      "home-hero-actions-after": () => h(Stats),
    });
  },
  enhanceApp({ app, router, siteData }) {},
};
