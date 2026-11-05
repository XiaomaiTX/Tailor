import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query"
import { fireEvent, render, screen } from "@testing-library/react"
import { createMemoryRouter, RouterProvider } from "react-router-dom"
import { routes, handleRouteError } from "@/app/router"
import { ApiProblemError } from "@/api/http"
import { ThemeProvider } from "@/components/theme-provider"
import { I18nProvider } from "@/i18n"
import { usePreferenceStore } from "@/stores/preference-store"

const mocks = vi.hoisted(() => ({ dashboard: vi.fn() }))

vi.mock("@/api/peer-relay", () => ({
  peerRelayApi: new Proxy(
    {},
    {
      get: (_target, property) =>
        property === "dashboard"
          ? mocks.dashboard
          : vi.fn().mockResolvedValue([]),
    }
  ),
}))

function renderApp(path = "/") {
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  const queryClient = new QueryClient({
    queryCache: new QueryCache({
      onError: (error) => handleRouteError(error, router),
    }),
    defaultOptions: { queries: { retry: false } },
  })

  const view = render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nProvider>
          <RouterProvider router={router} />
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
  return { ...view, router }
}

describe("App routing and authentication", () => {
  beforeEach(() => {
    usePreferenceStore.setState({ locale: "zh-CN" })
    window.history.replaceState({}, "", "/")
    mocks.dashboard.mockReset().mockRejectedValue(
      new ApiProblemError({
        type: "urn:go-react-starter:problem:unauthorized",
        title: "需要登录",
        status: 401,
      })
    )
  })

  it("shows the SSO login page when the dashboard session is missing", async () => {
    renderApp()

    expect(
      await screen.findByRole("link", { name: /使用 SSO 登录/ })
    ).toHaveAttribute("href", "/api/v1/auth/login")
  })

  it.each(["/requests", "/policy", "/settings"])(
    "redirects an expired session at %s to login",
    async (path) => {
      const { router } = renderApp(path)
      await screen.findByRole("link", { name: /使用 SSO 登录/ })
      expect(router.state.location.pathname).toBe("/login")
    }
  )

  it("shows not found for an unknown URL", async () => {
    const { router } = renderApp("/unknown")
    expect(router.state.location.pathname).toBe("/unknown")
    expect(screen.getByRole("heading", { level: 1 })).toBeInTheDocument()
    expect(mocks.dashboard).not.toHaveBeenCalled()
  })

  it.each(["/", "/requests", "/policy", "/settings", "/relays"])(
    "renders the allowed page or redirects a regular user entering %s",
    async (path) => {
      mocks.dashboard.mockResolvedValue({
        deviceCount: 0,
        relayCount: 0,
        selectableRelayCount: 0,
        assignmentCount: 0,
        pendingApplicationCount: 0,
        policyWritesEnabled: false,
        currentUser: {
          username: "alice",
          groups: [],
          isAdmin: false,
          isReviewer: false,
          canApply: false,
          canManageRelays: false,
          canEditAllDevices: false,
        },
      })
      const { router } = renderApp(path)

      expect((await screen.findAllByText("alice")).length).toBeGreaterThan(0)
      expect(screen.queryByRole("link", { name: /使用 SSO 登录/ })).toBeNull()
      expect(router.state.location.pathname).toBe(
        path === "/relays" ? "/relays" : "/"
      )
      fireEvent.click(screen.getByRole("button", { name: "Peer Relay" }))
      expect(
        await screen.findByRole("heading", { name: "Peer Relay 节点" })
      ).toBeInTheDocument()
      expect(router.state.location.pathname).toBe("/relays")
      expect(
        screen.getByRole("button", { name: "Peer Relay" })
      ).toHaveAttribute("aria-current", "page")
    }
  )
})
