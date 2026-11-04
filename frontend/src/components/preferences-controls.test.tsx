import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { LanguageToggle } from "@/components/language-toggle"
import { ModeToggle } from "@/components/mode-toggle"
import { ThemeProvider } from "@/components/theme-provider"
import { I18nProvider } from "@/i18n"
import { usePreferenceStore } from "@/stores/preference-store"

function Controls() {
  return (
    <ThemeProvider>
      <I18nProvider>
        <LanguageToggle />
        <ModeToggle />
      </I18nProvider>
    </ThemeProvider>
  )
}

describe("偏好切换控件", () => {
  beforeEach(() => {
    usePreferenceStore.setState({ locale: "zh-CN", theme: "light" })
  })

  it("切换语言并同步文档语言", async () => {
    const user = userEvent.setup()
    render(<Controls />)

    await user.click(screen.getByRole("button", { name: "选择语言" }))
    await user.click(await screen.findByText("英语"))

    expect(usePreferenceStore.getState().locale).toBe("en-US")
    expect(document.documentElement.lang).toBe("en-US")
  })

  it("从菜单切换主题", async () => {
    const user = userEvent.setup()
    render(<Controls />)

    await user.click(screen.getByRole("button", { name: "选择主题" }))
    await user.click(await screen.findByText("深色"))

    expect(usePreferenceStore.getState().theme).toBe("dark")
    expect(document.documentElement).toHaveClass("dark")
  })
})
