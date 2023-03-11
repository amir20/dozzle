import { createRequire } from "module";
import { defineConfig } from "vitepress";
import { createWriteStream } from "node:fs";
import { resolve } from "node:path";
import { SitemapStream, streamToPromise } from "sitemap";

const require = createRequire(import.meta.url);
const pkg = require("dozzle/package.json");

const links = [] as { url: string; lastmod?: number }[];

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
          { text: "Healthcheck", link: "/guide/healthcheck" },
          { text: "Remote Host", link: "/guide/remote-host" },
          { text: "Supported Env Vars", link: "/guide/supported-env-vars" },
        ],
      },
      {
        text: "Troubleshooting",
        items: [{ text: "FAQ", link: "/guide/faq" }],
      },
    ],

    footer: {
      message: "Released under the MIT License. Open sourced and sponsored by Docker OSS.",
      copyright: "Copyright © 2019-present <a href='https://amirraminfar.me'>Amir Raminfar</a>",
    },

    socialLinks: [{ icon: "github", link: "https://github.com/amir20/dozzle" }],
  },

  transformHtml: (_, id, { pageData }) => {
    if (!/[\\/]404\.html$/.test(id))
      links.push({
        // you might need to change this if not using clean urls mode
        url: pageData.relativePath.replace(/((^|\/)index)?\.md$/, "$2"),
        lastmod: pageData.lastUpdated,
      });
  },

  buildEnd: async ({ outDir }) => {
    const sitemap = new SitemapStream({
      hostname: "https://dozzle.dev/",
    });
    const writeStream = createWriteStream(resolve(outDir, "sitemap.xml"));
    sitemap.pipe(writeStream);
    links.forEach((link) => sitemap.write(link));
    const promise = streamToPromise(sitemap);
    sitemap.end();
    await promise;
  },
});
