# Shippy documentation

Source of the website and documentation published at <https://shippy.run/>.

Built with [VitePress](https://vitepress.dev/). All content is Markdown in this directory. This
README documents the site itself and is excluded from the build.

## Develop

Run everything from the `docs/` directory, or use the Makefile targets from the repository root.

```bash
cd docs
npm ci
npm run docs:dev      # http://localhost:5173/
```

```bash
make docs-dev         # same, from the repository root
```

## Build

```bash
npm run docs:build    # output: docs/.vitepress/dist
npm run docs:preview  # serve the build locally
make docs-build       # from the repository root
```

The build fails on dead internal links, so a green build is also a link check. Pull requests that
touch `docs/` run the build in CI.

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
│   ├── logo.svg                 # copy of images/logo.svg in the repository root
│   └── CNAME                    # custom domain: shippy.run
└── .vitepress/
    ├── config.ts                # nav, sidebar, SEO, search
    └── theme/                   # brand colours from the logo palette
```

The documentation mirrors the [repository README](../README.md), split into pages. Change both in
the same pull request when behaviour or options change.

## Deployment

Pushing to `main` with changes under `docs/` runs `.github/workflows/docs.yml`, which builds the
site and publishes it to GitHub Pages at <https://shippy.run/>.

**Settings → Pages → Build and deployment → Source** must stay on **GitHub Actions**. Changing the
custom domain in that screen silently resets the source back to *Deploy from a branch*, which
publishes the repository root through Jekyll and 404s every built asset. Check the source after
touching the domain.

The domain is pinned in three places that have to agree: `docs/public/CNAME`, and `base` /
`sitemap.hostname` in `docs/.vitepress/config.ts`.
