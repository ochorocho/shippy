import { defineConfig } from 'vitepress'

const base = '/'
const hostname = 'https://shippy.run/'
const repo = 'https://github.com/ochorocho/shippy'

export default defineConfig({
  title: 'Shippy',
  description: 'Zero-downtime deployments for Composer-based PHP projects.',
  base,
  cleanUrls: true,
  lastUpdated: true,
  sitemap: { hostname },

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: `${base}logo.svg` }],
    ['meta', { name: 'theme-color', content: '#FF8700' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'Shippy — deployments for PHP projects' }],
    [
      'meta',
      {
        property: 'og:description',
        content:
          'A single Go binary for zero-downtime, atomic deployments of Composer-based PHP projects.',
      },
    ],
    ['meta', { property: 'og:url', content: hostname }],
    ['meta', { property: 'og:image', content: `${hostname}logo.svg` }],
    ['meta', { name: 'twitter:card', content: 'summary' }],
  ],

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'Shippy',

    nav: [
      { text: 'Guide', link: '/guide/', activeMatch: '/guide/' },
      { text: 'Reference', link: '/reference/commands', activeMatch: '/reference/' },
      {
        text: 'v0.2.4',
        items: [
          { text: 'Releases', link: `${repo}/releases` },
          { text: 'Changelog', link: `${repo}/releases/latest` },
          { text: 'Contributing', link: '/guide/contributing' },
        ],
      },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is Shippy?', link: '/guide/' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
          ],
        },
        {
          text: 'Using Shippy',
          items: [
            { text: 'Configuration', link: '/guide/configuration' },
            { text: 'File Selection', link: '/guide/file-selection' },
            { text: 'Deployment Commands', link: '/guide/deployment-commands' },
            { text: 'SSH Connections', link: '/guide/ssh' },
            { text: 'Deployment Process', link: '/guide/deployment-process' },
            { text: 'CI/CD', link: '/guide/ci-cd' },
            { text: 'Example Configurations', link: '/guide/examples' },
          ],
        },
        {
          text: 'Reference',
          items: [
            { text: 'Commands', link: '/reference/commands' },
            { text: 'Configuration Options', link: '/reference/configuration' },
          ],
        },
        {
          text: 'Project',
          items: [{ text: 'Contributing', link: '/guide/contributing' }],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Commands', link: '/reference/commands' },
            { text: 'Configuration Options', link: '/reference/configuration' },
          ],
        },
        {
          text: 'Guide',
          items: [
            { text: 'What is Shippy?', link: '/guide/' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Configuration', link: '/guide/configuration' },
            { text: 'File Selection', link: '/guide/file-selection' },
            { text: 'Deployment Commands', link: '/guide/deployment-commands' },
            { text: 'SSH Connections', link: '/guide/ssh' },
            { text: 'Deployment Process', link: '/guide/deployment-process' },
            { text: 'CI/CD', link: '/guide/ci-cd' },
            { text: 'Example Configurations', link: '/guide/examples' },
          ],
        },
      ],
    },

    socialLinks: [{ icon: 'github', link: repo }],

    search: { provider: 'local' },

    outline: { level: [2, 3], label: 'On this page' },

    editLink: {
      pattern: 'https://github.com/ochorocho/shippy-website/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 Jochen Roth',
    },

    docFooter: { prev: 'Previous', next: 'Next' },
  },
})
