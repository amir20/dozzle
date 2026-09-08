// The sidebar shape lives here once. Locales only supply labels, so adding a
// page means one slug in this file plus one label per locale, not five
// hand-maintained sidebar trees that drift apart.

// A plain slug is one page. `slug` is a page that owns sub-pages, so the group
// header stays clickable. `group` is a heading with no page behind it, which is
// what most groupings want: it files related pages together without inventing a
// landing page that would need translating five times.
export type Item = string | { slug: string; items: string[] } | { group: string; items: string[] };

export type Section = { key: string; items: Item[] };

export const SECTIONS: Section[] = [
  { key: "introduction", items: ["what-is-dozzle", "getting-started"] },
  { key: "platforms", items: ["swarm-mode", "k8s", "podman"] },
  {
    key: "cloud",
    items: [
      "dozzle-cloud",
      "dozzle-cloud/connecting",
      "dozzle-cloud/channels",
      "dozzle-cloud/plans",
      "dozzle-cloud/your-data",
    ],
  },
  {
    key: "advanced",
    items: [
      {
        slug: "authentication",
        items: ["authentication/simple", "authentication/oauth", "authentication/forward-proxy"],
      },
      { group: "containers", items: ["container-names", "container-groups", "container-links", "app-icons"] },
      { group: "hosts", items: ["remote-hosts", "agent", "hostname"] },
      { group: "control", items: ["actions", "shell"] },
      { group: "logs", items: ["sql-engine", "log-files-on-disk"] },
      { group: "tools", items: ["dtop", "mcp"] },
      "alerts-and-webhooks",
      "changing-base",
      "filters",
      "default-profile",
      "healthcheck",
      "analytics",
    ],
  },
  { key: "troubleshooting", items: ["faq", "debugging", "supported-env-vars"] },
];

// Pages that live at the root of a locale rather than under /guide/.
export const ABOUT = ["team", "support"];

export type Labels = {
  label: string;
  lang: string;
  description: string;
  nav: { home: string; guide: string; cloud: string; releases: string; newIssue: string };
  sections: Record<string, string>;
  groups: Record<string, string>;
  pages: Record<string, string>;
  footer: { message: string; copyright: string };
  ui: {
    outline: string;
    darkModeSwitch: string;
    returnToTop: string;
    lastUpdated: string;
    docFooterPrev: string;
    docFooterNext: string;
    editLink: string;
    sidebarMenu: string;
  };
  search: {
    buttonText: string;
    buttonAriaLabel: string;
    noResults: string;
    resetButton: string;
    footerNavigate: string;
    footerSelect: string;
    footerClose: string;
  };
};

const COPYRIGHT_LINK = "<a href='https://amirraminfar.me'>Amir Raminfar</a>";

export function link(base: string, path: string) {
  return `${base}${path}`;
}

export function buildThemeConfig(base: string, t: Labels, version: string) {
  return {
    logo: "/logo.svg",
    nav: [
      { text: t.nav.home, link: link(base, "/") },
      { text: t.nav.guide, link: link(base, "/guide/what-is-dozzle"), activeMatch: `${base}/guide/` },
      { text: t.nav.cloud, link: "https://cloud.dozzle.dev" },
      {
        text: `v${version}`,
        items: [
          { text: t.nav.releases, link: "https://github.com/amir20/dozzle/releases" },
          { text: t.nav.newIssue, link: "https://github.com/amir20/dozzle/issues/new/choose" },
        ],
      },
    ],
    sidebar: [
      ...SECTIONS.map((section) => ({
        text: t.sections[section.key],
        items: section.items.map((item) => {
          const page = (slug: string) => ({ text: t.pages[slug], link: link(base, `/guide/${slug}`) });
          if (typeof item === "string") return page(item);
          // collapsed:true keeps the section tidy; VitePress still opens the group
          // automatically when the active page is inside it.
          const children = item.items.map(page);
          return "slug" in item
            ? { ...page(item.slug), collapsed: true, items: children }
            : { text: t.groups[item.group], collapsed: true, items: children };
        }),
      })),
      {
        text: t.sections.about,
        items: ABOUT.map((slug) => ({ text: t.pages[slug], link: link(base, `/${slug}`) })),
      },
    ],
    editLink: {
      pattern: "https://github.com/amir20/dozzle/edit/master/docs/:path",
      text: t.ui.editLink,
    },
    footer: {
      message: t.footer.message,
      copyright: t.footer.copyright.replace("{link}", COPYRIGHT_LINK),
    },
    outline: { label: t.ui.outline },
    darkModeSwitchLabel: t.ui.darkModeSwitch,
    returnToTopLabel: t.ui.returnToTop,
    sidebarMenuLabel: t.ui.sidebarMenu,
    lastUpdatedText: t.ui.lastUpdated,
    docFooter: { prev: t.ui.docFooterPrev, next: t.ui.docFooterNext },
  };
}
