// https://vitepress.dev/guide/custom-theme
import { h } from "vue";
import DefaultTheme from "vitepress/theme";
import { Icon } from "@iconify/vue";

import "@fontsource-variable/playfair-display";
import "./style.css";
import HeroDemo from "./components/HeroDemo.vue";
import BuyMeCoffee from "./components/BuyMeCoffee.vue";
import Stats from "./components/Stats.vue";
import HeroTrust from "./components/HeroTrust.vue";
import Supported from "./components/Supported.vue";
import InstallCommand from "./components/InstallCommand.vue";
import WhyDozzle from "./components/WhyDozzle.vue";
import Testimonials from "./components/Testimonials.vue";
import FinalCta from "./components/FinalCta.vue";
import SponsoredBy from "./components/SponsoredBy.vue";

export default {
  ...DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      "home-hero-image": () => h(HeroDemo),
      "sidebar-nav-after": () => h(BuyMeCoffee),
      "home-hero-actions-after": () => [h(Stats), h(HeroTrust)],
      "home-hero-after": () => [h(InstallCommand), h(Supported)],
      "home-features-after": () => [h(WhyDozzle), h(Testimonials), h(FinalCta), h(SponsoredBy)],
    });
  },
  enhanceApp(ctx) {
    DefaultTheme.enhanceApp(ctx);
    ctx.app.component("Icon", Icon);
  },
};
