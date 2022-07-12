module.exports = {
  theme: 'cosmos',
  title: 'Haqq Documentation',
  locales: {
    '/': {
      lang: 'en-US'
    },
  },
  markdown: {
    extendMarkdown: (md) => {
      md.use(require("markdown-it-katex"));
    },
  },
  head: [
    [
      "link",
      {
        rel: "stylesheet",
        href:
          "https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.5.1/katex.min.css",
      },
    ],
    [
      "link",
      {
        rel: "stylesheet",
        href:
          "https://cdn.jsdelivr.net/github-markdown-css/2.2.1/github-markdown.css",
      },
    ],
  ],
  base: process.env.VUEPRESS_BASE || '/',
  plugins: [
    'vuepress-plugin-element-tabs'
  ],
  head: [
    ['link', { rel: "apple-touch-icon", sizes: "180x180", href: "/apple-touch-icon.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "32x32", href: "/favicon-32x32.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "16x16", href: "/favicon-16x16.png" }],
    ['link', { rel: "manifest", href: "/site.webmanifest" }],
    ['meta', { name: "msapplication-TileColor", content: "#2e3148" }],
    ['meta', { name: "theme-color", content: "#ffffff" }],
    ['link', { rel: "icon", type: "image/svg+xml", href: "/favicon.ico" }],
    // ['link', { rel: "apple-touch-icon-precomposed", href: "/apple-touch-icon-precomposed.png" }],
  ],
  themeConfig: {
    repo: 'haqq-network/haqq',
    docsRepo: 'haqq-network/haqq',
    docsBranch: 'master',
    docsDir: 'docs',
    editLinks: true,
    custom: true,
    project: {
      name: 'Haqq',
      denom: 'ISLM',
      ticker: 'ISLM',
      binary: 'haqqd',
      testnet_denom: 'ISLM',
      testnet_ticker: 'ISLM',
      rpc_ws_url: 'https://rpc-ws.eth.haqq.network:443/',
      rpc_eth_url: 'https://rpc.eth.haqq.network:443',
      rpc_tm_url: 'https://rpc.tm.haqq.network:443',
      grpc_cosmos_url: 'https://grpc.cosmos.haqq.network:443',
      rest_api_url: 'https://rest.cosmos.haqq.network:443',
      rpc_url_local: 'http://localhost:8545/',
      chain_id: '11235',
      testnet_chain_id: '112357',
      latest_version: 'v1.0.0',
      version_number: '1',
      testnet_version_number: '1',
      block_explorer_url: 'https://explorer.haqq.network/',
    },
    logo: {
      src: '/haqq1.svg',
    },
    topbar: {
      banner: false
    },
    sidebar: {
      auto: false,
      nav: [
        {
          title: 'Reference',
          children: [
            {
              title: 'Introduction',
              directory: true,
              path: '/intro'
            },
            {
              title: 'Quick Start',
              directory: true,
              path: '/quickstart'
            },
            {
              title: 'Basics',
              directory: true,
              path: '/basics'
            },
            {
              title: 'Core Concepts',
              directory: true,
              path: '/core'
            },
          ]
        },
        {
          title: 'Guides',
          children: [
            {
              title: 'Localnet',
              directory: true,
              path: '/guides/localnet'
            },
            {
              title: 'Keys and Wallets',
              directory: true,
              path: '/guides/keys-wallets'
            },
            {
              title: 'Ethereum Tooling',
              directory: true,
              path: '/guides/tools'
            },
            {
              title: 'Validators',
              directory: true,
              path: '/guides/validators'
            },
            {
              title: 'Upgrades',
              directory: true,
              path: '/guides/upgrades'
            },
            {
              title: 'Key Management System',
              directory: true,
              path: '/guides/kms'
            },
          ]
        },
        {
          title: 'APIs',
          children: [
            {
              title: 'JSON-RPC',
              directory: true,
              path: '/api/json-rpc'
            },
            {
              title: 'Protobuf Reference',
              directory: false,
              path: '/api/proto-docs'
            },
          ]
        },
        {
          title: 'Testnet',
          children: [
            {
              title: 'Join Testnet',
              directory: false,
              path: '/testnet/join'
            },
            {
              title: 'Token Faucet',
              directory: false,
              path: '/testnet/faucet'
            },
            {
              title: 'Deploy Node on Cloud',
              directory: false,
              path: '/testnet/cloud_providers'
            }
          ]
        },
        {
          title: 'Specifications',
          children: [{
            title: 'Modules',
            directory: true,
            path: '/modules'
          }]
        },
        {
          title: 'Block Explorers',
          children: [
            {
              title: 'Block Explorers',
              path: '/tools/explorers'
            },
            {
              title: 'Haqq explorer',
              path: 'https://explorer.haqq.network/'
            },
          ]
        },
        {
          title: 'Resources',
          children: [
            {
              title: 'Ethermint Library API Reference',
              path: 'https://pkg.go.dev/github.com/tharsis/ethermint'
            },
            {
              title: 'JSON-RPC API Reference',
              path: '/api/json-rpc/endpoints'
            }
          ]
        }
      ]
    },
    footer: {
      logo: '/haqq1.svg',
      textLink: {
        //text: 'Haqq Network',
        url: 'https://docs.haqq.network'
      },
      services: [
        {
          service: 'github',
          url: 'https://github.com/haqq-network'
        },
      ],
      smallprint: 'This website is maintained by Haqq Network.',
      links: [{
        title: 'Documentation',
        children: [{
          title: 'Cosmos SDK Docs',
          url: 'https://docs.cosmos.network/master/'
        },
        {
          title: 'Ethereum Docs',
          url: 'https://ethereum.org/developers'
        },
        {
          title: 'Tendermint Core Docs',
          url: 'https://docs.tendermint.com'
        }
        ]
      },
      ]
    },
    versions: [
      {
        "label": "main",
        "key": "main"
      },
    ],
  }
};
