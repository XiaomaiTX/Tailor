import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { useI18n } from "@/i18n"
import { ChevronsUpDown, LogOut } from "lucide-react"

export function NavUser({
  name,
  email,
  role,
  isLoggingOut,
  onLogout,
}: {
  name: string
  email?: string
  role: string
  isLoggingOut: boolean
  onLogout: () => void
}) {
  const { isMobile } = useSidebar()
  const { t } = useI18n()
  const initials = name.slice(0, 2).toLocaleUpperCase()

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenuTrigger>
          <SidebarMenuButton
            size="lg"
            tooltip={name}
            aria-label={name}
            className="aria-expanded:bg-sidebar-accent aria-expanded:text-sidebar-accent-foreground"
          >
            <Avatar className="rounded-none after:rounded-none">
              <AvatarFallback className="rounded-none bg-sidebar-primary text-sidebar-primary-foreground">
                {initials || "?"}
              </AvatarFallback>
            </Avatar>
            <span className="grid min-w-0 flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{name}</span>
              <span className="truncate text-xs text-sidebar-foreground/70">
                {email || role}
              </span>
            </span>
            <ChevronsUpDown className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
          </SidebarMenuButton>
          <DropdownMenu
            className="w-64"
            placement={isMobile ? "bottom end" : "right bottom"}
            offset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="p-0 font-normal">
                <span className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <Avatar className="rounded-none after:rounded-none">
                    <AvatarFallback className="rounded-none bg-sidebar-primary text-sidebar-primary-foreground">
                      {initials || "?"}
                    </AvatarFallback>
                  </Avatar>
                  <span className="grid min-w-0 flex-1 leading-tight">
                    <span className="truncate font-medium">{name}</span>
                    {email ? (
                      <span className="truncate text-xs text-muted-foreground">
                        {email}
                      </span>
                    ) : null}
                    <span className="truncate text-xs text-muted-foreground">
                      {role}
                    </span>
                  </span>
                </span>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              variant="destructive"
              isDisabled={isLoggingOut}
              textValue={t("auth.logout")}
              onAction={onLogout}
            >
              <LogOut aria-hidden="true" />
              {isLoggingOut ? t("relay.sidebar.signingOut") : t("auth.logout")}
            </DropdownMenuItem>
          </DropdownMenu>
        </DropdownMenuTrigger>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
