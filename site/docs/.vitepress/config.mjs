import { defineConfig } from 'vitepress'

// 项目页部署在 https://attson.github.io/atstarter/,base 必须与仓库名一致,
// 否则静态资源 404。
export default defineConfig({
  base: '/atstarter/',
  lang: 'zh-CN',
  title: 'AT Starter',
  description: '本地项目快速启动器(Wails v2 + Vue3 桌面 App)',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '使用文档', link: '/guide/' },
      { text: 'FAQ', link: '/guide/faq' },
      { text: '下载', link: 'https://github.com/attson/atstarter/releases/latest' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '使用文档',
          items: [
            { text: '介绍与使用', link: '/guide/' },
            { text: 'FAQ / 故障排查', link: '/guide/faq' },
          ],
        },
      ],
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/attson/atstarter' },
    ],
    search: { provider: 'local' },
  },
})
