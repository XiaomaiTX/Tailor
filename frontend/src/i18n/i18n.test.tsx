import { render, screen } from "@testing-library/react"
import { I18nProvider, useI18n } from "@/i18n"
import { usePreferenceStore } from "@/stores/preference-store"

function Probe() {
  const { t } = useI18n()
  return <span>{t("app.home")}</span>
}

describe("i18n", () => {
  beforeEach(() => {
    usePreferenceStore.setState({ locale: "zh-CN" })
  })

  it("根据持久化语言选择词典", () => {
    usePreferenceStore.setState({ locale: "en-US" })
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>
    )
    expect(screen.getByText("Console")).toBeInTheDocument()
    expect(document.documentElement.lang).toBe("en-US")
    expect(document.title).toBe("Tailor")
  })

  it("缺失键返回键名", () => {
    function MissingKeyProbe() {
      const { t } = useI18n()
      return <span>{t("missing.value")}</span>
    }

    render(
      <I18nProvider>
        <MissingKeyProbe />
      </I18nProvider>
    )
    expect(screen.getByText("missing.value")).toBeInTheDocument()
  })
})
