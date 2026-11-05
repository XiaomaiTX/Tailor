import { render, screen } from "@testing-library/react"
import { ThemeProvider } from "@/components/theme-provider"
import { I18nProvider } from "@/i18n"
import { usePreferenceStore } from "@/stores/preference-store"
import { OidcLoginPage } from "./oidc-login-page"

describe("OidcLoginPage", () => {
  beforeEach(() => {
    usePreferenceStore.setState({ locale: "zh-CN" })
    window.history.replaceState({}, "", "/login")
  })

  it("shows one SSO entry that starts the OIDC flow", () => {
    render(
      <ThemeProvider>
        <I18nProvider>
          <OidcLoginPage />
        </I18nProvider>
      </ThemeProvider>
    )

    expect(screen.getByRole("link", { name: /使用 SSO 登录/ })).toHaveAttribute(
      "href",
      "/api/v1/auth/login"
    )
    expect(screen.getAllByRole("link", { name: /使用 SSO 登录/ })).toHaveLength(1)
  })

  it("shows a generic message when the SSO callback fails", () => {
    window.history.replaceState({}, "", "/login?error=sso")
    render(
      <ThemeProvider>
        <I18nProvider>
          <OidcLoginPage />
        </I18nProvider>
      </ThemeProvider>
    )
    expect(screen.getByRole("alert")).toHaveTextContent("SSO 登录失败")
  })
})
