import { defineConfig } from "vitepress";

const navEN = [
  { text: "Guide", link: "/guide/overview" }
];

const navZH = [
  { text: "指南", link: "/zh/guide/overview" }
];

const sidebarEN = [
  {
    text: "Guide",
    items: [{ text: "Overview", link: "/guide/overview" }]
  }
];

const sidebarZH = [
  {
    text: "指南",
    items: [{ text: "总览", link: "/zh/guide/overview" }]
  }
];

export default defineConfig({
  title: "Goyais Docs",
  description: "Goyais engineering documentation and platform/runtime notes",
  lastUpdated: true,
  cleanUrls: true,
  themeConfig: {
    logo: "/logo.svg",
    socialLinks: [{ icon: "github", link: "https://github.com/GoyacJ/Goyais" }],
    footer: {
      message: "Released under MIT License",
      copyright: "Copyright © 2026 Goyais Contributors"
    }
  },
  locales: {
    root: {
      label: "English",
      lang: "en-US",
      themeConfig: {
        nav: navEN,
        sidebar: sidebarEN
      }
    },
    zh: {
      label: "简体中文",
      lang: "zh-CN",
      link: "/zh/",
      themeConfig: {
        nav: navZH,
        sidebar: sidebarZH
      }
    }
  }
});
