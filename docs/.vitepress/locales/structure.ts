// The sidebar shape lives here once. Locales only supply labels, so adding a
// page means one slug in this file plus one label per locale, not five
// hand-maintained sidebar trees that drift apart.

// A plain slug is one page. The object form is a page that owns sub-pages, which
// VitePress renders as a clickable group that expands when you are inside it.
export type Item = string | { slug: string; items: string[] };

export type Section = { key: string; items: Item[] };

export const SECTIONS: Section[] = [
  { key: "introduction", items: ["what-is-dozzle", "getting-started", "dtop"] },
  { key: "platforms", items: ["swarm-mode", "k8s", "podman"] },
  { key: "notifications", items: ["alerts-and-webhooks", "dozzle-cloud"] },
  {
    key: "advanced",
    items: [
      {
        slug: "authentication",
        items: ["authentication/simple", "authentication/oauth", "authentication/forward-proxy"],
      },
      "actions",
      "app-icons",
      "shell",
      "mcp",
      "agent",
      "changing-base",
      "container-names",
      "container-groups",
      "container-links",
      "analytics",
      "default-profile",
      "hostname",
      "filters",
      "healthcheck",
      "remote-hosts",
      "log-files-on-disk",
      "sql-engine",
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
          return { ...page(item.slug), collapsed: true, items: item.items.map(page) };
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
