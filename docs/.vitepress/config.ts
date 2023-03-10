import { defineConfig } from "vitepress";

export default defineConfig({
  lang: "en-US",
  title: "Dozzle",
  description: "A lightweight, open-source, and secure log viewer for Docker.",

  lastUpdated: true,
  cleanUrls: true,

  head: [
    [
      "script",
      {
        async: "",
        src: "https://www.googletagmanager.com/gtag/js?id=G-X3Z4496XFK",
      },
    ],
    [
      "script",
      {},
      `function gtag(){dataLayer.push(arguments)}window.dataLayer=window.dataLayer||[],gtag("js",new Date),gtag("config","G-X3Z4496XFK")`,
    ],
  ],
  themeConfig: {
    editLink: {
      pattern: "https://github.com/amir20/dozzle/edit/master/docs/:path",
    },
    nav: [
      { text: "Home", link: "/" },
      { text: "Guide", link: "/guide/what-is-dozzle" },
    ],
    sidebar: [
      {
        text: "Introduction",
        items: [
          { text: "What is Dozzle?", link: "/guide/what-is-dozzle" },
          { text: "Getting Started", link: "/guide/getting-started" },
        ],
      },
    ],

    footer: {
      message: "Released under the MIT License. Open sourced and sponsored by Docker OSS.",
      copyright: "Copyright © 2019-present <a href='https://amirraminfar.me'>Amir Raminfar</a>",
    },

    socialLinks: [{ icon: "github", link: "https://github.com/amir20/dozzle" }],
  },
});
