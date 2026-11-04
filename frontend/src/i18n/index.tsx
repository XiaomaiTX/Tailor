/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  type PropsWithChildren,
} from "react"
import enUS from "@/i18n/messages/en-US"
import zhCN from "@/i18n/messages/zh-CN"
import { usePreferenceStore, type Locale } from "@/stores/preference-store"

const dictionaries = { "zh-CN": zhCN, "en-US": enUS } as const
type I18nContextValue = {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string) => string
}

const I18nContext = createContext<I18nContextValue | undefined>(undefined)

function translate(dictionary: Record<string, unknown>, key: string): string {
  const value = key.split(".").reduce<unknown>((source, part) => {
    if (typeof source !== "object" || source === null) return undefined
    return (source as Record<string, unknown>)[part]
  }, dictionary)
  return typeof value === "string" ? value : key
}

export function I18nProvider({ children }: PropsWithChildren) {
  const locale = usePreferenceStore((state) => state.locale)
  const setLocale = usePreferenceStore((state) => state.setLocale)
  const title = translate(dictionaries[locale], "app.title")

  useEffect(() => {
    document.documentElement.lang = locale
    document.title = title
  }, [locale, title])

  const value = useMemo(
    () => ({
      locale,
      setLocale,
      t: (key: string) => translate(dictionaries[locale], key),
    }),
    [locale, setLocale]
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const context = useContext(I18nContext)
  if (!context) {
    throw new Error("useI18n must be used within an I18nProvider")
  }

  return context
}
