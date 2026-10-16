# shippy-website

Marketing site and documentation for [Shippy](https://github.com/ochorocho/shippy), the deployment
tool for Composer-based PHP projects.

Built with [VitePress](https://vitepress.dev/). All content is Markdown under `docs/`.

## Develop

```bash
npm install
npm run docs:dev      # http://localhost:5173/
```

## Build

```bash
npm run docs:build    # output: docs/.vitepress/dist
npm run docs:preview  # serve the build locally
```

The build fails on dead internal links, so a green build is also a link check.

## Structure

```
docs/
├── index.md                     # landing page (layout: home)
├── guide/                       # narrative documentation
│   ├── index.md                 # what is Shippy?
│   ├── installation.md
│   ├── quick-start.md
│   ├── configuration.md
│   ├── file-selection.md        # deny-by-default allowlist
│   ├── deployment-commands.md   # only/except, command_context
│   ├── ssh.md
│   ├── deployment-process.md
│   ├── ci-cd.md
│   ├── examples.md
│   └── contributing.md
├── reference/                   # lookup tables
│   ├── commands.md
│   └── configuration.md
├── public/
│   ├── logo.svg                 # from ochorocho/shippy, images/logo.svg
│   └── CNAME                    # custom domain: shippy.run
└── .vitepress/
    ├── config.ts                # nav, sidebar, SEO, search
    └── theme/                   # brand colours from the logo palette
```

The documentation content mirrors the
[README of ochorocho/shippy](https://github.com/ochorocho/shippy/blob/main/README.md), split into
pages. When that README changes, update the corresponding page here.

## Deployment

Pushing to `main` runs `.github/workflows/deploy.yml`, which builds the site and publishes it to
GitHub Pages at <https://shippy.run/>.

**Settings → Pages → Build and deployment → Source** must stay on **GitHub Actions**. Changing the
custom domain in that screen silently resets the source back to *Deploy from a branch*, which
publishes the repository root through Jekyll and 404s every built asset. Check the source after
touching the domain.

The domain is pinned in three places that have to agree: `docs/public/CNAME`, and `base` /
`sitemap.hostname` in `docs/.vitepress/config.ts`.

## License

MIT, matching the upstream project.
