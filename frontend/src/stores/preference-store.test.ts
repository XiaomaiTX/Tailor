import { usePreferenceStore } from "@/stores/preference-store"

describe("preference store", () => {
  beforeEach(() => {
    localStorage.clear()
    usePreferenceStore.setState({ locale: "zh-CN", theme: "system" })
  })

  it("仅持久化主题与语言", () => {
    usePreferenceStore.getState().setLocale("en-US")
    usePreferenceStore.getState().setTheme("dark")

    const stored = JSON.parse(
      localStorage.getItem("go-react-starter-preferences") || "{}"
    ) as {
      state?: Record<string, unknown>
    }
    expect(stored.state).toEqual({ locale: "en-US", theme: "dark" })
  })
})
