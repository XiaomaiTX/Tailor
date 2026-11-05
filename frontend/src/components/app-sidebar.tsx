import type * as React from "react"
import { RadioTower, type LucideIcon } from "lucide-react"
import type { RelayUser } from "@/api/generated/model"
import { NavMain } from "@/components/nav-main"
import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarRail,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar"
import { useI18n } from "@/i18n"

type NavItem<TPageKey extends string> = {
  key: TPageKey
  label: string
  icon: LucideIcon
  badge?: number
}

type AppSidebarProps<TPageKey extends string> = React.ComponentProps<
  typeof Sidebar
> & {
  items: NavItem<TPageKey>[]
  activePage: TPageKey
  currentUser?: RelayUser
  syncError?: string
  isLoggingOut: boolean
  onNavigate: (page: TPageKey) => void
  onLogout: () => void
}

export function AppSidebar<TPageKey extends string>({
  items,
  activePage,
  currentUser,
  syncError,
  isLoggingOut,
  onNavigate,
  onLogout,
  ...props
}: AppSidebarProps<TPageKey>) {
  const { t } = useI18n()
  const { isMobile, setOpenMobile } = useSidebar()
  const navigateToPage = (page: TPageKey) => {
    onNavigate(page)
    if (isMobile) setOpenMobile(false)
  }
  const primaryItems = items.filter(
    (item) => item.key !== "policy" && item.key !== "settings"
  )
  const managementItems = items.filter(
    (item) => item.key === "policy" || item.key === "settings"
  )
  const role = currentUser?.isAdmin
    ? t("relay.common.admin")
    : currentUser?.isReviewer
      ? t("relay.common.reviewer")
      : t("relay.common.member")

  return (
    <Sidebar
      collapsible="icon"
      mobileTitle={t("relay.sidebar.mobileTitle")}
      mobileDescription={t("relay.sidebar.mobileDescription")}
      mobileCloseLabel={t("relay.sidebar.close")}
      {...props}
    >
      <SidebarHeader className="border-b border-sidebar-border">
        <div className="flex min-w-0 items-center gap-2 px-2 py-1.5">
          <span
            role="img"
            aria-label={`${t("relay.brand")} ${t("relay.product")}`}
            className="flex size-8 shrink-0 items-center justify-center bg-sidebar-primary text-sidebar-primary-foreground"
          >
            <RadioTower className="size-4" aria-hidden="true" />
          </span>
          <span className="grid min-w-0 flex-1 leading-tight group-data-[collapsible=icon]:hidden">
            <span className="truncate text-[10px] font-bold tracking-[0.18em] text-sidebar-primary">
              {t("relay.brand")}
            </span>
            <span className="truncate text-sm font-semibold">
              {t("relay.product")}
            </span>
          </span>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup className="pb-1 group-data-[collapsible=icon]:hidden">
          <SidebarGroupLabel>{t("relay.environment")}</SidebarGroupLabel>
          <div className="flex min-w-0 items-center gap-2 px-3 py-1 text-xs text-sidebar-foreground/75">
            <span
              className={`size-1.5 shrink-0 ${syncError ? "bg-destructive" : "bg-sidebar-primary"}`}
              aria-hidden="true"
            />
            <span className="truncate">{currentUser?.username ?? "—"}</span>
            <span className="ml-auto shrink-0 text-[10px]">
              {syncError
                ? t("relay.overview.disconnected")
                : t("relay.overview.connected")}
            </span>
          </div>
        </SidebarGroup>

        <nav
          aria-label={t("relay.header.system")}
          className="flex min-h-0 flex-1 flex-col"
        >
          <NavMain
            items={primaryItems}
            activePage={activePage}
            groupLabel={t("relay.common.workspace")}
            onNavigate={navigateToPage}
          />
          {managementItems.length > 0 ? (
            <NavMain
              items={managementItems}
              activePage={activePage}
              groupLabel={t("relay.sidebar.administration")}
              onNavigate={navigateToPage}
            />
          ) : null}
        </nav>
      </SidebarContent>

      <SidebarFooter>
        <SidebarSeparator className="mx-0" />
        <NavUser
          name={currentUser?.username ?? t("relay.sidebar.loadingUser")}
          email={currentUser?.email}
          role={role}
          isLoggingOut={isLoggingOut}
          onLogout={onLogout}
        />
      </SidebarFooter>
      <SidebarRail
        aria-label={t("relay.sidebar.toggle")}
        title={t("relay.sidebar.toggle")}
      />
    </Sidebar>
  )
}
