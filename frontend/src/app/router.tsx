import { createBrowserRouter } from "react-router-dom"
import App from "@/App"
import { NotFoundPage } from "@/pages/not-found-page"

export const router = createBrowserRouter([
  { path: "/", Component: App },
  { path: "*", Component: NotFoundPage },
])
