// https://vitepress.dev/guide/custom-theme
import { h } from "vue";
import DefaultTheme from "vitepress/theme";

import "@fontsource-variable/playfair-display";
import "./style.css";
import HeroVideo from "./components/HeroVideo.vue";
import BuyMeCoffee from "./components/BuyMeCoffee.vue";
import Stats from "./components/Stats.vue";

export default {
  ...DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      "home-hero-image": () => h(HeroVideo),
      "sidebar-nav-after": () => h(BuyMeCoffee),
      "home-hero-actions-after": () => h(Stats),
    });
  },
  enhanceApp(ctx) {
    DefaultTheme.enhanceApp(ctx);
  },
};
