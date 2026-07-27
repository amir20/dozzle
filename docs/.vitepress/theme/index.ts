// https://vitepress.dev/guide/custom-theme
import { h } from "vue";
import DefaultTheme from "vitepress/theme";
import { Icon } from "@iconify/vue";

import "@fontsource-variable/playfair-display";
import "./style.css";
import HeroVideo from "./components/HeroVideo.vue";
import BuyMeCoffee from "./components/BuyMeCoffee.vue";
import Stats from "./components/Stats.vue";
import Supported from "./components/Supported.vue";
import InstallCommand from "./components/InstallCommand.vue";
import CloudPromo from "./components/CloudPromo.vue";
import SponsoredBy from "./components/SponsoredBy.vue";

export default {
  ...DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      "home-hero-image": () => h(HeroVideo),
      "sidebar-nav-after": () => h(BuyMeCoffee),
      "home-hero-actions-after": () => h(Stats),
      "home-hero-after": () => [h(InstallCommand), h(Supported)],
      "home-features-after": () => [h(CloudPromo), h(SponsoredBy)],
    });
  },
  enhanceApp(ctx) {
    DefaultTheme.enhanceApp(ctx);
    ctx.app.component("Icon", Icon);
  },
};
