/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  type PropsWithChildren,
} from "react"
import { usePreferenceStore, type ThemeMode } from "@/stores/preference-store"

type ThemeProviderState = {
  theme: ThemeMode
  setTheme: (theme: ThemeMode) => void
}

const COLOR_SCHEME_QUERY = "(prefers-color-scheme: dark)"
const ThemeProviderContext = createContext<ThemeProviderState | undefined>(
  undefined
)

function getSystemTheme() {
  return window.matchMedia?.(COLOR_SCHEME_QUERY).matches ? "dark" : "light"
}

function applyTheme(theme: ThemeMode) {
  const resolvedTheme = theme === "system" ? getSystemTheme() : theme
  document.documentElement.classList.remove("light", "dark")
  document.documentElement.classList.add(resolvedTheme)
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const theme = usePreferenceStore((state) => state.theme)
  const setTheme = usePreferenceStore((state) => state.setTheme)

  useEffect(() => {
    applyTheme(theme)
    if (theme !== "system" || !window.matchMedia) {
      return undefined
    }

    const mediaQuery = window.matchMedia(COLOR_SCHEME_QUERY)
    const handleChange = () => applyTheme("system")
    mediaQuery.addEventListener("change", handleChange)

    return () => mediaQuery.removeEventListener("change", handleChange)
  }, [theme])

  const value = useMemo(() => ({ theme, setTheme }), [setTheme, theme])
  return (
    <ThemeProviderContext.Provider value={value}>
      {children}
    </ThemeProviderContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeProviderContext)
  if (!context) {
    throw new Error("useTheme must be used within a ThemeProvider")
  }

  return context
}
