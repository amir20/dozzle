import { defineConfig } from 'vitepress'


export default defineConfig({
  lang: 'en-US',
  title: 'Dozzle',
  description: 'A lightweight, open-source, and secure log viewer for Docker.',

  lastUpdated: true,
  cleanUrls: true,

  themeConfig: {
    logo: '/logo.svg',
    editLink: {
      pattern: 'https://github.com/amir20/dozzle/edit/master/docs/:path'
    },
    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Introduction', link: '/guide/what-is-dozzle' },
          { text: 'Getting Started', link: '/guide/getting-started' },
        ]
      }
    ]
  }
})
