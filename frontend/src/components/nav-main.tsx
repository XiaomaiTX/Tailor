import type { LucideIcon } from "lucide-react"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

type NavItem<TPageKey extends string> = {
  key: TPageKey
  label: string
  icon: LucideIcon
  badge?: number
}

export function NavMain<TPageKey extends string>({
  items,
  activePage,
  groupLabel,
  onNavigate,
}: {
  items: NavItem<TPageKey>[]
  activePage: TPageKey
  groupLabel: string
  onNavigate: (page: TPageKey) => void
}) {
  return (
    <SidebarGroup>
      <SidebarGroupLabel>{groupLabel}</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => {
          const Icon = item.icon
          const isActive = activePage === item.key

          return (
            <SidebarMenuItem key={item.key}>
              <SidebarMenuButton
                isActive={isActive}
                tooltip={item.label}
                aria-label={item.label}
                aria-current={isActive ? "page" : undefined}
                onPress={() => onNavigate(item.key)}
              >
                <Icon aria-hidden="true" />
                <span className="group-data-[collapsible=icon]:hidden">
                  {item.label}
                </span>
                {item.badge ? (
                  <SidebarMenuBadge>{item.badge}</SidebarMenuBadge>
                ) : null}
              </SidebarMenuButton>
            </SidebarMenuItem>
          )
        })}
      </SidebarMenu>
    </SidebarGroup>
  )
}
