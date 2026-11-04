import { create } from "zustand"
import { createJSONStorage, persist } from "zustand/middleware"

export type Locale = "zh-CN" | "en-US"
export type ThemeMode = "light" | "dark" | "system"

interface PreferenceState {
  locale: Locale
  theme: ThemeMode
  setLocale: (locale: Locale) => void
  setTheme: (theme: ThemeMode) => void
}

export const usePreferenceStore = create<PreferenceState>()(
  persist(
    (set) => ({
      locale: "zh-CN",
      theme: "system",
      setLocale: (locale) => set({ locale }),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: "go-react-starter-preferences",
      storage: createJSONStorage(() => localStorage),
      partialize: ({ locale, theme }) => ({ locale, theme }),
    }
  )
)
