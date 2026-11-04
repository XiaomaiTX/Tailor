export default {
  app: { title: "Go React Starter", home: "首页" },
  home: {
    title: "Go React Starter",
    body: "一个可扩展的 Go 与 React 技术基座。",
  },
  notFound: {
    title: "页面不存在",
    body: "该地址没有匹配的页面。",
    back: "返回首页",
  },
  preferences: {
    language: "语言",
    languageMenu: "选择语言",
    chinese: "简体中文",
    english: "英语",
    theme: "主题",
    themeMenu: "选择主题",
    light: "浅色",
    dark: "深色",
    system: "跟随系统",
  },
  errors: {
    validation: "请求内容不合法。",
    notFound: "资源不存在。",
    internal: "服务暂时不可用，请稍后再试。",
    unknown: "请求失败，请稍后再试。",
  },
} as const
