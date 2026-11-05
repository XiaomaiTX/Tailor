import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { createMemoryRouter, RouterProvider } from "react-router-dom"
import { ApiProblemError } from "@/api/http"
import { ThemeProvider } from "@/components/theme-provider"
import { I18nProvider } from "@/i18n"
import { usePreferenceStore } from "@/stores/preference-store"
import {
  PeerRelayConsole,
  PeerRelayRoutePage,
} from "./peer-relay-console"

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

function renderConsole(queryClient: QueryClient) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <PeerRelayConsole />,
        children: [
          {
            index: true,
            handle: { peerRelayPage: "overview" },
            element: <PeerRelayRoutePage page="overview" />,
          },
        ],
      },
    ],
    { initialEntries: ["/"] }
  )

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nProvider>
          <RouterProvider router={router} />
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

describe("PeerRelayConsole authentication", () => {
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
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    renderConsole(queryClient)

    expect(
      await screen.findByRole("link", { name: /使用 SSO 登录/ })
    ).toHaveAttribute("href", "/api/v1/auth/login")
  })

  it("loads the console when the returned session is accepted by the API", async () => {
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
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })

    renderConsole(queryClient)

    expect((await screen.findAllByText("alice")).length).toBeGreaterThan(0)
    expect(screen.queryByRole("link", { name: /使用 SSO 登录/ })).toBeNull()
  })
})
