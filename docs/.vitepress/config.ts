import { createRequire } from "module";
import { defineConfig } from "vitepress";

const require = createRequire(import.meta.url);
const pkg = require("dozzle/package.json");

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
    logo: "/logo.svg",
    search: {
      provider: "local",
    },
    editLink: {
      pattern: "https://github.com/amir20/dozzle/edit/master/docs/:path",
    },
    nav: [
      { text: "Home", link: "/" },
      { text: "Guide", link: "/guide/what-is-dozzle", activeMatch: "/guide/" },
      {
        text: `v${pkg.version}`,
        items: [
          {
            text: "Releases",
            link: "https://github.com/amir20/dozzle/releases",
          },
          {
            text: "New Issue",
            link: "https://github.com/amir20/dozzle/issues/new/choose",
          },
        ],
      },
    ],
    sidebar: [
      {
        text: "Introduction",
        items: [
          { text: "What is Dozzle?", link: "/guide/what-is-dozzle" },
          { text: "Getting Started", link: "/guide/getting-started" },
        ],
      },
      {
        text: "Advanced Configuration",
        items: [
          { text: "Authentication", link: "/guide/authentication" },
          { text: "Actions", link: "/guide/actions" },
          { text: "Agent Mode", link: "/guide/agent" },
          { text: "Changing Base", link: "/guide/changing-base" },
          { text: "Container Names", link: "/guide/container-names" },
          { text: "Container Groups", link: "/guide/container-groups" },
          { text: "Data Analytics", link: "/guide/analytics" },
          { text: "Display Name", link: "/guide/hostname" },
          { text: "Filters", link: "/guide/filters" },
          { text: "Healthcheck", link: "/guide/healthcheck" },
          { text: "Remote Hosts", link: "/guide/remote-hosts" },
          { text: "Swarm Mode", link: "/guide/swarm-mode" },
          { text: "Supported Env Vars", link: "/guide/supported-env-vars" },
          { text: "Logging Files on Disk", link: "/guide/log-files-on-disk" },
          { text: "SQL Engine", link: "/guide/sql-engine" },
        ],
      },
      {
        text: "Troubleshooting",
        items: [
          { text: "FAQ", link: "/guide/faq" },
          { text: "Debugging", link: "/guide/debugging" },
        ],
      },
      {
        text: "About",
        items: [
          { text: "Team", link: "/team" },
          { text: "Support", link: "/support" },
        ],
      },
    ],

    footer: {
      message: "Released under the MIT License. Open sourced and sponsored by Docker OSS.",
      copyright: "Copyright © 2019-present <a href='https://amirraminfar.me'>Amir Raminfar</a>",
    },

    socialLinks: [
      { icon: "github", link: "https://github.com/amir20/dozzle" },
      {
        icon: {
          svg: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="currentColor" d="M21.81 10.25c-.06-.04-.56-.43-1.64-.43c-.28 0-.56.03-.84.08c-.21-1.4-1.38-2.11-1.43-2.14l-.29-.17l-.18.27c-.24.36-.43.77-.51 1.19c-.2.8-.08 1.56.33 2.21c-.49.28-1.29.35-1.46.35H2.62c-.34 0-.62.28-.62.63c0 1.15.18 2.3.58 3.38c.45 1.19 1.13 2.07 2 2.61c.98.6 2.59.94 4.42.94c.79 0 1.61-.07 2.42-.22c1.12-.2 2.2-.59 3.19-1.16A8.3 8.3 0 0 0 16.78 16c1.05-1.17 1.67-2.5 2.12-3.65h.19c1.14 0 1.85-.46 2.24-.85c.26-.24.45-.53.59-.87l.08-.24l-.19-.14m-17.96.99h1.76c.08 0 .16-.07.16-.16V9.5c0-.08-.07-.16-.16-.16H3.85c-.09 0-.16.07-.16.16v1.58c.01.09.07.16.16.16m2.43 0h1.76c.08 0 .16-.07.16-.16V9.5c0-.08-.07-.16-.16-.16H6.28c-.09 0-.16.07-.16.16v1.58c.01.09.07.16.16.16m2.47 0h1.75c.1 0 .17-.07.17-.16V9.5c0-.08-.06-.16-.17-.16H8.75c-.08 0-.15.07-.15.16v1.58c0 .09.06.16.15.16m2.44 0h1.77c.08 0 .15-.07.15-.16V9.5c0-.08-.06-.16-.15-.16h-1.77c-.08 0-.15.07-.15.16v1.58c0 .09.07.16.15.16M6.28 9h1.76c.08 0 .16-.09.16-.18V7.25c0-.09-.07-.16-.16-.16H6.28c-.09 0-.16.06-.16.16v1.57c.01.09.07.18.16.18m2.47 0h1.75c.1 0 .17-.09.17-.18V7.25c0-.09-.06-.16-.17-.16H8.75c-.08 0-.15.06-.15.16v1.57c0 .09.06.18.15.18m2.44 0h1.77c.08 0 .15-.09.15-.18V7.25c0-.09-.07-.16-.15-.16h-1.77c-.08 0-.15.06-.15.16v1.57c0 .09.07.18.15.18m0-2.28h1.77c.08 0 .15-.07.15-.16V5c0-.1-.07-.17-.15-.17h-1.77c-.08 0-.15.06-.15.17v1.56c0 .08.07.16.15.16m2.46 4.52h1.76c.09 0 .16-.07.16-.16V9.5c0-.08-.07-.16-.16-.16h-1.76c-.08 0-.15.07-.15.16v1.58c0 .09.07.16.15.16"/></svg>`,
        },
        link: "https://hub.docker.com/r/amir20/dozzle",
      },
    ],
  },

  sitemap: {
    hostname: "https://dozzle.dev/",
  },
});
