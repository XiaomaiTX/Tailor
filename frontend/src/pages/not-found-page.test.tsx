import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { I18nProvider } from "@/i18n"
import { NotFoundPage } from "@/pages/not-found-page"
import { usePreferenceStore } from "@/stores/preference-store"

describe("NotFoundPage", () => {
  beforeEach(() => {
    usePreferenceStore.setState({ locale: "zh-CN" })
  })

  it("显示本地化 404 页面并提供返回首页链接", () => {
    render(
      <MemoryRouter>
        <I18nProvider>
          <NotFoundPage />
        </I18nProvider>
      </MemoryRouter>
    )

    expect(
      screen.getByRole("heading", { name: "页面不存在" })
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "返回首页" })).toHaveAttribute(
      "href",
      "/"
    )
  })
})
