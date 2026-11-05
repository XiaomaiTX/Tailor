import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query"
import { RouterProvider } from "react-router-dom"

import { ApiProblemError } from "@/api/http"
import { router, handleRouteError } from "@/app/router"
import { ThemeProvider } from "@/components/theme-provider"
import { I18nProvider } from "@/i18n"
import "./index.css"

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => handleRouteError(error),
  }),
  defaultOptions: {
    queries: {
      retry: (failureCount, error) =>
        !(error instanceof ApiProblemError && error.problem.status === 401) &&
        failureCount < 1,
    },
  },
})

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nProvider>
          <RouterProvider router={router} />
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </StrictMode>
)
