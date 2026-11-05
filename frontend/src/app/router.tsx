import { createBrowserRouter } from "react-router-dom"
import App from "@/App"
import { OidcLoginPage } from "@/features/auth/oidc-login-page"
import { NotFoundPage } from "@/pages/not-found-page"

export const router = createBrowserRouter([
  { path: "/", Component: App },
  { path: "/login", Component: OidcLoginPage },
  { path: "*", Component: NotFoundPage },
])
