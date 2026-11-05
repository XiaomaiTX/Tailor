import { createBrowserRouter } from "react-router-dom"
import App from "@/App"
import { OidcLoginPage } from "@/features/auth/oidc-login-page"
import {
  PeerRelayRoutePage,
  type PeerRelayPageKey,
} from "@/features/peer-relay/peer-relay-console"
import { NotFoundPage } from "@/pages/not-found-page"

const consolePages: PeerRelayPageKey[] = [
  "overview",
  "editor",
  "relays",
  "requests",
  "policy",
  "settings",
]

export const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: consolePages.map((page) => {
      const route = {
        element: <PeerRelayRoutePage page={page} />,
        handle: { peerRelayPage: page },
      }
      return page === "overview"
        ? { ...route, index: true as const }
        : { ...route, path: page }
    }),
  },
  { path: "/login", Component: OidcLoginPage },
  { path: "*", Component: NotFoundPage },
])
