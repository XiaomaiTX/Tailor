import type { RelayDashboard } from "@/api/generated/model"
import { ApiProblemError } from "@/api/http"
import { peerRelayApi } from "@/api/peer-relay"
import App from "@/App"
import { OidcLoginPage } from "@/features/auth/oidc-login-page"
import { ApplicationsRoute } from "@/features/peer-relay/pages/applications-page"
import { OverviewRoute } from "@/features/peer-relay/pages/overview-page"
import { PolicyRoute } from "@/features/peer-relay/pages/policy-page"
import { RelaysRoute } from "@/features/peer-relay/pages/relays-page"
import { SettingsRoute } from "@/features/peer-relay/pages/settings-page"
import type { PeerRelayPageKey } from "@/features/peer-relay/queries"
import { NotFoundPage } from "@/pages/not-found-page"
import { useCallback } from "react"
import {
  createBrowserRouter,
  redirect,
  useMatches,
  useNavigate,
  type RouteObject,
} from "react-router-dom"

export function usePageNavigation() {
  const navigate = useNavigate()
  return useCallback(
    (page: PeerRelayPageKey, state?: { nodeID: string }) => {
      void navigate(page === "overview" ? "/" : `/${page}`, { state })
    },
    [navigate]
  )
}

function pageAccessLoader(page: PeerRelayPageKey) {
  if (page !== "requests" && page !== "policy" && page !== "settings") {
    return undefined
  }

  return async () => {
    try {
      const dashboard = await peerRelayApi.dashboard()
      const user = dashboard.currentUser
      const canAccess =
        page === "requests" ? user.canApply || user.isReviewer : user.isAdmin

      return canAccess ? dashboard : redirect("/")
    } catch (error) {
      if (error instanceof ApiProblemError) {
        return redirect(error.problem.status === 401 ? "/login" : "/")
      }
      throw error
    }
  }
}

// Route components stay with the route configuration; this module is not a refresh boundary.
// eslint-disable-next-line react-refresh/only-export-components
function AppRoute() {
  const matches = useMatches()
  const navigate = useNavigate()

  const page = matches
    .map(
      (match) =>
        (match.handle as { peerRelayPage?: PeerRelayPageKey } | undefined)
          ?.peerRelayPage
    )
    .find((routePage): routePage is PeerRelayPageKey => routePage !== undefined)
  const initialDashboard = matches
    .map((match) => match.data as RelayDashboard | undefined)
    .find((data) => data?.currentUser !== undefined)

  const navigateToPage = usePageNavigation()
  const navigateToLogin = useCallback(() => {
    void navigate("/login", { replace: true })
  }, [navigate])

  return (
    <App
      page={page ?? "overview"}
      initialDashboard={initialDashboard}
      onNavigate={navigateToPage}
      onNavigateToLogin={navigateToLogin}
    />
  )
}

export const routes: RouteObject[] = [
  {
    path: "/",
    Component: AppRoute,
    children: [
      {
        index: true,
        Component: OverviewRoute,
        handle: { peerRelayPage: "overview" },
      },
      {
        path: "relays",
        Component: RelaysRoute,
        handle: { peerRelayPage: "relays" },
      },
      {
        path: "requests",
        Component: ApplicationsRoute,
        handle: { peerRelayPage: "requests" },
        loader: pageAccessLoader("requests"),
      },
      {
        path: "policy",
        Component: PolicyRoute,
        handle: { peerRelayPage: "policy" },
        loader: pageAccessLoader("policy"),
      },
      {
        path: "settings",
        Component: SettingsRoute,
        handle: { peerRelayPage: "settings" },
        loader: pageAccessLoader("settings"),
      },
      {
        path: "editor",
        handle: { peerRelayPage: "editor" },
        lazy: async () => ({
          Component: (await import("@/features/peer-relay/pages/editor-page"))
            .EditorRoute,
        }),
      },
    ],
  },
  { path: "/login", Component: OidcLoginPage },
  { path: "*", Component: NotFoundPage },
]

export const router = createBrowserRouter(routes)

export function handleRouteError(error: unknown, targetRouter = router) {
  if (
    error instanceof ApiProblemError &&
    error.problem.status === 401 &&
    targetRouter.state.location.pathname !== "/login"
  ) {
    void targetRouter.navigate("/login")
  }
}
