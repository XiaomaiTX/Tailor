export default {
  app: { title: "Go React Starter", home: "Home" },
  home: {
    title: "Go React Starter",
    body: "An extensible Go and React technical foundation.",
  },
  notFound: {
    title: "Page not found",
    body: "No page matches this address.",
    back: "Back home",
  },
  preferences: {
    language: "Language",
    languageMenu: "Choose language",
    chinese: "Chinese",
    english: "English",
    theme: "Theme",
    themeMenu: "Choose theme",
    light: "Light",
    dark: "Dark",
    system: "System",
  },
  errors: {
    validation: "The request is invalid.",
    notFound: "The resource does not exist.",
    internal: "The service is temporarily unavailable.",
    unknown: "The request failed. Please try again.",
  },
} as const
