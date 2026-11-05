import type { RelayDashboard } from "@/api/generated/model"
import { ApiProblemError } from "@/api/http"
import { peerRelayApi } from "@/api/peer-relay"
import { AppSidebar } from "@/components/app-sidebar"
import { LanguageToggle } from "@/components/language-toggle"
import { ModeToggle } from "@/components/mode-toggle"
import { Badge } from "@/components/ui/badge"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { errorMessage } from "@/features/peer-relay/utils"
import { useI18n } from "@/i18n"
import { cn } from "@/lib/utils"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Activity,
  Bell,
  GitBranch,
  RadioTower,
  RefreshCw,
  SlidersHorizontal,
  TriangleAlert,
  Waypoints,
  type LucideIcon,
} from "lucide-react"

import { Toast } from "@/features/peer-relay/components/page-heading"
import {
  useDashboard,
  type PeerRelayPageKey,
} from "@/features/peer-relay/queries"
import { useToast } from "@/features/peer-relay/use-toast"
import { Outlet } from "react-router-dom"
export default function App({
  page,
  initialDashboard,
  onNavigate,
  onNavigateToLogin,
}: {
  page: PeerRelayPageKey
  initialDashboard?: RelayDashboard
  onNavigate: (page: PeerRelayPageKey) => void
  onNavigateToLogin: () => void
}) {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const dashboardQuery = useDashboard(initialDashboard)
  const dashboard = dashboardQuery.data
  const currentUser = dashboard?.currentUser
  const isAdmin = currentUser?.isAdmin ?? false
  const isReviewer = currentUser?.isReviewer ?? false
  const { toast, setToast } = useToast()
  const syncMutation = useMutation({
    mutationFn: peerRelayApi.sync,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["peer-relay"] })
      setToast(t("relay.header.synced"))
    },
  })
  const logoutMutation = useMutation({
    mutationFn: peerRelayApi.logout,
    onSuccess: () => {
      queryClient.clear()
      onNavigateToLogin()
    },
  })

  const navItems: {
    key: PeerRelayPageKey
    label: string
    icon: LucideIcon
    badge?: number
  }[] = [
    { key: "overview", label: t("relay.nav.overview"), icon: Activity },
    { key: "editor", label: t("relay.nav.editor"), icon: Waypoints },
    { key: "relays", label: t("relay.nav.relays"), icon: RadioTower },
    ...(currentUser?.canApply || isReviewer
      ? [
          {
            key: "requests" as const,
            label: t("relay.nav.requests"),
            icon: Bell,
            badge: isReviewer ? dashboard?.pendingApplicationCount : undefined,
          },
        ]
      : []),
    ...(isAdmin
      ? [
          {
            key: "policy" as const,
            label: t("relay.nav.policy"),
            icon: GitBranch,
          },
        ]
      : []),
    ...(isAdmin
      ? [
          {
            key: "settings" as const,
            label: t("relay.nav.settings"),
            icon: SlidersHorizontal,
          },
        ]
      : []),
  ]

  const pageTitle =
    navItems.find((item) => item.key === page)?.label ?? t("relay.nav.overview")
  const apiError = dashboardQuery.error
  const configError =
    apiError instanceof ApiProblemError && apiError.problem.status === 503
  return (
    <SidebarProvider className="min-h-svh bg-sidebar text-foreground">
      <AppSidebar
        items={navItems}
        activePage={page}
        currentUser={currentUser}
        syncError={dashboard?.syncError}
        isLoggingOut={logoutMutation.isPending}
        onNavigate={onNavigate}
        onLogout={() => logoutMutation.mutate()}
        variant="inset"
      />
      <SidebarInset className="min-h-svh min-w-0">
        <header className="sticky top-0 z-20 border-b border-border bg-background/90 px-5 backdrop-blur-xl sm:px-8 lg:px-10">
          <div className="flex h-[72px] items-center justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
              <SidebarTrigger
                aria-label={t("relay.sidebar.toggle")}
                className="shrink-0"
              />
              <Separator orientation="vertical" className="h-6" />
              <Breadcrumb
                aria-label={t("relay.sidebar.breadcrumb")}
                className="min-w-0"
              >
                <BreadcrumbList className="flex-nowrap">
                  <BreadcrumbItem>
                    <BreadcrumbPage className="truncate">
                      {pageTitle}
                    </BreadcrumbPage>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            </div>
            <div className="flex shrink-0 items-center gap-2 sm:gap-3">
              <LanguageToggle />
              <ModeToggle />
              <Badge
                variant={dashboard?.syncError ? "destructive" : "outline"}
                className="hidden sm:inline-flex"
              >
                {dashboard?.syncError
                  ? t("relay.header.stale")
                  : t("relay.header.synced")}
              </Badge>
              {isAdmin ? (
                <Button
                  variant="outline"
                  className="flex h-9 items-center gap-2 px-3 text-xs tracking-normal normal-case"
                  onPress={() => syncMutation.mutate()}
                  isDisabled={syncMutation.isPending}
                >
                  <RefreshCw
                    size={14}
                    className={cn(syncMutation.isPending && "animate-spin")}
                  />
                  <span className="hidden sm:inline">
                    {syncMutation.isPending
                      ? t("relay.header.syncing")
                      : t("relay.header.refresh")}
                  </span>
                </Button>
              ) : null}
            </div>
          </div>
        </header>

        <div className="w-full px-5 py-7 sm:px-8 sm:py-9 lg:px-10">
          {apiError ? (
            <div className="mb-6 flex items-start gap-3 rounded-none border border-border bg-muted px-4 py-3.5 text-muted-foreground">
              <TriangleAlert size={18} className="mt-0.5 shrink-0" />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold">
                  {configError
                    ? t("relay.overview.configTitle")
                    : errorMessage(apiError, t("errors.unknown"))}
                </p>
                <p className="mt-1 text-xs leading-5 opacity-75">
                  {configError
                    ? t("relay.overview.configBody")
                    : t("relay.overview.disconnected")}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon-sm"
                onPress={() => dashboardQuery.refetch()}
                aria-label={t("relay.common.retry")}
              >
                <RefreshCw size={15} />
              </Button>
            </div>
          ) : null}

          <Outlet />
        </div>
      </SidebarInset>
      <Toast message={toast} />
    </SidebarProvider>
  )
}
