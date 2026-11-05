import type { Device, RelayDashboard } from "@/api/generated/model"
import { peerRelayApi } from "@/api/peer-relay"
import { useEffect } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
export type PeerRelayPageKey =
  "overview" | "editor" | "relays" | "requests" | "policy" | "settings"
const EMPTY_DEVICES: Device[] = []

const queryKeys = {
  dashboard: ["peer-relay", "dashboard"] as const,
  devices: ["peer-relay", "devices"] as const,
  relays: ["peer-relay", "relays"] as const,
  applications: ["peer-relay", "applications"] as const,
  canvasGraph: ["peer-relay", "canvas-graph"] as const,
  selectorRules: ["peer-relay", "selector-rules"] as const,
  revisions: ["peer-relay", "revisions"] as const,
  audits: ["peer-relay", "audits"] as const,
  settings: ["peer-relay", "settings"] as const,
}

export function useDashboard(initialDashboard?: RelayDashboard) {
  const queryClient = useQueryClient()
  useEffect(() => {
    if (initialDashboard)
      queryClient.setQueryData(queryKeys.dashboard, initialDashboard)
  }, [initialDashboard, queryClient])
  return useQuery({
    queryKey: queryKeys.dashboard,
    queryFn: peerRelayApi.dashboard,
    initialData: initialDashboard,
    refetchInterval: 60_000,
  })
}
export function usePeerRelayData() {
  const queryClient = useQueryClient()
  const dashboardQuery = useDashboard()
  const dashboard = dashboardQuery.data
  const currentUser = dashboard?.currentUser
  const isAdmin = currentUser?.isAdmin ?? false
  const isReviewer = currentUser?.isReviewer ?? false
  const canApply = currentUser?.canApply ?? false
  const canManageRelays = currentUser?.canManageRelays ?? false
  const canEditAllDevices = currentUser?.canEditAllDevices ?? false

  const devicesQuery = useQuery({
    queryKey: [...queryKeys.devices, canEditAllDevices],
    queryFn: () => peerRelayApi.devices(),
    enabled: Boolean(currentUser),
  })
  const devices = devicesQuery.data ?? EMPTY_DEVICES
  const relaysQuery = useQuery({
    queryKey: [...queryKeys.relays, canManageRelays],
    queryFn: () => peerRelayApi.relays(canManageRelays),
    enabled: Boolean(currentUser),
  })
  const relays = relaysQuery.data ?? []
  const applicationsQuery = useQuery({
    queryKey: queryKeys.applications,
    queryFn: () => peerRelayApi.applications(),
    enabled: Boolean(
      currentUser && (currentUser.canApply || currentUser.isReviewer)
    ),
  })
  const refreshAll = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
      queryClient.invalidateQueries({ queryKey: queryKeys.devices }),
      queryClient.invalidateQueries({ queryKey: queryKeys.relays }),
      queryClient.invalidateQueries({ queryKey: queryKeys.applications }),
      queryClient.invalidateQueries({ queryKey: queryKeys.canvasGraph }),
      queryClient.invalidateQueries({ queryKey: queryKeys.selectorRules }),
    ])
  }

  return {
    queryClient,
    dashboardQuery,
    dashboard,
    currentUser,
    isAdmin,
    isReviewer,
    canApply,
    canManageRelays,
    canEditAllDevices,
    devicesQuery,
    devices,
    relaysQuery,
    relays,
    applicationsQuery,
    refreshAll,
  }
}
export { queryKeys }
