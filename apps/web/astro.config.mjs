import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import react from '@astrojs/react';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  site: 'https://triage.algotyrnt.com',
  output: 'static',
  vite: {
    plugins: [tailwindcss()],
  },
  integrations: [
    react(),
    starlight({
      title: 'triage docs',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/algotyrnt/triage',
        },
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Overview', link: '/docs/overview/' },
            { label: 'Quickstart (5 Mins)', link: '/docs/quickstart/' },
          ],
        },
        {
          label: 'Go SDK',
          items: [
            { label: 'SDK Integration Guide', link: '/docs/sdk/' },
            { label: 'Telemetry & Architecture', link: '/docs/sdk-architecture/' },
            { label: 'Monorepos & Multi-Module', link: '/docs/monorepo-support/' },
          ],
        },
        {
          label: 'Engine & Diagnostics',
          items: [
            { label: 'AST Engine & Node Slicing', link: '/docs/ast-engine/' },
            { label: 'Gemini AI Diagnostics', link: '/docs/gemini-ai/' },
            { label: 'GitHub App & Issue Automation', link: '/docs/github-integration/' },
          ],
        },
        {
          label: 'Operations & Reference',
          items: [
            { label: 'Authentication & Team RBAC', link: '/docs/team-and-rbac/' },
            { label: 'Self-Hosting & Docker', link: '/docs/self-hosting/' },
            { label: 'Environment & Configuration', link: '/docs/configuration/' },
            { label: 'Development & Releases', link: '/docs/development/' },
            { label: 'Engine REST API', link: '/docs/api-reference/' },
            { label: 'Troubleshooting & FAQ', link: '/docs/troubleshooting/' },
          ],
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});
