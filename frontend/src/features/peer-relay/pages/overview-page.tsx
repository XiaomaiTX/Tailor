import {
  Activity,
  ArrowRight,
  ArrowUpRight,
  Bell,
  Check,
  ChevronRight,
  Circle,
  Link2,
  LockKeyhole,
  Radio,
  RadioTower,
  RefreshCw,
  Server,
  Waypoints,
} from "lucide-react"
import type { Device, Relay, RelayApplication } from "@/api/generated/model"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DeviceIcon,
  EmptyState,
  MetricCard,
  StatusTag,
} from "@/features/peer-relay/components/view-primitives"
import { formatDate } from "@/features/peer-relay/utils"
import { cn } from "@/lib/utils"

export function OverviewPage({
  dashboard,
  devices,
  relays,
  applications,
  isAdmin,
  isLoading,
  locale,
  t,
  onOpenEditor,
  onOpenApplications,
  onSync,
  syncPending,
}: {
  dashboard?: {
    deviceCount: number
    relayCount: number
    selectableRelayCount: number
    assignmentCount: number
    pendingApplicationCount: number
    lastSyncedAt?: string | null
    syncError?: string
    policyHash?: string
    policyWritesEnabled: boolean
  }
  devices: Device[]
  relays: Relay[]
  applications: RelayApplication[]
  isAdmin: boolean
  isLoading: boolean
  locale: string
  t: (key: string) => string
  onOpenEditor: (deviceId?: string) => void
  onOpenApplications: () => void
  onSync: () => void
  syncPending: boolean
}) {
  const featured = devices.slice(0, 4)
  return (
    <div className="space-y-6">
      <section className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <MetricCard
          icon={Server}
          label={t("relay.overview.devices")}
          value={isLoading ? "—" : (dashboard?.deviceCount ?? 0)}
          detail={t("relay.common.allDevices")}
        />
        <MetricCard
          icon={RadioTower}
          label={t("relay.overview.relays")}
          value={isLoading ? "—" : (dashboard?.relayCount ?? 0)}
          detail={t("relay.overview.selectable")}
          detailValue={dashboard?.selectableRelayCount ?? 0}
        />
        <MetricCard
          icon={Link2}
          label={t("relay.overview.assignments")}
          value={isLoading ? "—" : (dashboard?.assignmentCount ?? 0)}
        />
        <Button
          variant="outline"
          className="h-auto min-h-[132px] flex-col items-stretch justify-start border-border bg-card p-4 text-left tracking-normal normal-case hover:bg-card"
          onPress={onOpenApplications}
        >
          <div className="flex items-center justify-between">
            <span className="grid size-9 place-items-center rounded-none bg-muted text-muted-foreground">
              <Bell size={17} />
            </span>
            <ChevronRight size={16} className="text-muted-foreground" />
          </div>
          <div className="mt-4 flex items-baseline justify-between gap-2">
            <span className="text-[27px] font-semibold tracking-[-0.04em]">
              {isLoading ? "—" : (dashboard?.pendingApplicationCount ?? 0)}
            </span>
            <span className="truncate text-[11px] text-muted-foreground">
              {t("relay.overview.pending")}
            </span>
          </div>
        </Button>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_minmax(300px,.8fr)]">
        <div className="overflow-hidden rounded-none border border-border bg-card">
          <div className="flex flex-col justify-between gap-4 border-b border-border px-5 py-5 sm:flex-row sm:items-center sm:px-6">
            <div>
              <div className="flex items-center gap-2">
                <span className="grid size-8 place-items-center rounded-none bg-muted text-muted-foreground">
                  <Waypoints size={16} />
                </span>
                <h3 className="text-sm font-semibold">
                  {t("relay.editor.title")}
                </h3>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                {t("relay.editor.routeNotice")}
              </p>
            </div>
            <Button
              className="flex items-center justify-center gap-2 px-4"
              onPress={() => onOpenEditor()}
            >
              {t("relay.overview.openEditor")} <ArrowRight size={14} />
            </Button>
          </div>
          <div className="grid gap-3 p-4 sm:grid-cols-2 sm:p-5">
            {featured.map((device) => {
              const deviceRelays = relays
                .filter((relay) => relay.selectable)
                .slice(0, 2)
              return (
                <Button
                  key={device.nodeId}
                  variant="outline"
                  className="group h-auto justify-start border-border p-4 text-left tracking-normal normal-case hover:bg-muted"
                  onPress={() => onOpenEditor(device.nodeId)}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-3">
                      <DeviceIcon tags={device.tags} />
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold">
                          {device.givenName || device.hostname}
                        </p>
                        <p className="mt-1 truncate font-mono text-[10px] text-muted-foreground">
                          {device.ipv4 || device.ipv6 || device.hostname}
                        </p>
                      </div>
                    </div>
                    <ChevronRight
                      size={15}
                      className="mt-1 shrink-0 text-muted-foreground transition group-hover:translate-x-0.5 group-hover:text-muted-foreground"
                    />
                  </div>
                  <div className="mt-4 flex items-center gap-2 pl-1">
                    <div className="grid size-5 place-items-center rounded-full border border-border bg-muted text-muted-foreground">
                      <Circle size={7} fill="currentColor" />
                    </div>
                    <div className="h-px w-6 bg-muted" />
                    {deviceRelays.length > 0 ? (
                      deviceRelays.map((relay) => (
                        <span
                          key={relay.nodeId}
                          className="flex items-center gap-1 rounded-full bg-muted px-2 py-1 text-[9px] text-muted-foreground"
                        >
                          <Radio size={10} />
                          {relay.displayName}
                        </span>
                      ))
                    ) : (
                      <span className="text-[10px] text-muted-foreground">
                        {t("relay.editor.noRelays")}
                      </span>
                    )}
                  </div>
                </Button>
              )
            })}
            {featured.length === 0 ? (
              <EmptyState title={t("relay.editor.noDevice")} icon={Server} />
            ) : null}
          </div>
        </div>

        <aside className="space-y-6">
          <div className="rounded-none border border-border bg-card p-5 sm:p-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="grid size-8 place-items-center rounded-none bg-muted text-muted-foreground">
                  <Activity size={16} />
                </span>
                <h3 className="text-sm font-semibold">
                  {t("relay.overview.connection")}
                </h3>
              </div>
              <Badge
                variant={dashboard?.syncError ? "destructive" : "secondary"}
              >
                {dashboard?.syncError
                  ? t("relay.overview.disconnected")
                  : t("relay.overview.connected")}
              </Badge>
            </div>
            <div className="mt-5 rounded-none bg-muted p-4">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">
                  {t("relay.overview.lastSync")}
                </span>
                <span className="font-medium">
                  {dashboard?.lastSyncedAt
                    ? formatDate(dashboard.lastSyncedAt, locale)
                    : t("relay.overview.never")}
                </span>
              </div>
              <div className="mt-3 flex items-center justify-between text-xs">
                <span className="text-muted-foreground">
                  {t("relay.overview.policyHash")}
                </span>
                <span className="font-mono text-[10px]">
                  {dashboard?.policyHash
                    ? `${dashboard.policyHash.slice(0, 10)}…`
                    : "—"}
                </span>
              </div>
            </div>
            <div className="mt-4 flex items-center gap-2 text-[11px]">
              {dashboard?.policyWritesEnabled ? (
                <Check size={14} className="text-primary" />
              ) : (
                <LockKeyhole size={13} className="text-muted-foreground" />
              )}
              <span className="text-muted-foreground">
                {dashboard?.policyWritesEnabled
                  ? t("relay.overview.writeOn")
                  : t("relay.overview.writeOff")}
              </span>
            </div>
            {isAdmin ? (
              <Button
                variant="outline"
                onPress={onSync}
                isDisabled={syncPending}
                className="mt-4 flex w-full items-center justify-center gap-2 tracking-normal normal-case"
              >
                <RefreshCw
                  size={13}
                  className={cn(syncPending && "animate-spin")}
                />
                {syncPending
                  ? t("relay.header.syncing")
                  : t("relay.header.refresh")}
              </Button>
            ) : null}
          </div>

          <div className="rounded-none border border-border bg-card p-5 sm:p-6">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-[10px] font-bold tracking-[0.15em] text-muted-foreground">
                  {t("relay.overview.recent")}
                </p>
                <h3 className="mt-1 text-sm font-semibold">
                  {t("relay.applications.title")}
                </h3>
              </div>
              {applications.length > 0 ? (
                <Button
                  variant="link"
                  onPress={onOpenApplications}
                  className="h-auto p-0 text-xs tracking-normal normal-case"
                >
                  {applications.length}{" "}
                  <ArrowUpRight size={12} className="inline" />
                </Button>
              ) : null}
            </div>
            {applications.length ? (
              <div className="mt-4 space-y-3">
                {applications.slice(0, 3).map((application) => (
                  <div
                    key={application.id}
                    className="flex items-center gap-3 border-t border-border pt-3 first:border-0 first:pt-0"
                  >
                    <DeviceIcon tags={application.device.tags} small />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium">
                        {application.displayName}
                      </p>
                      <p className="mt-1 truncate text-[10px] text-muted-foreground">
                        {application.device.givenName ||
                          application.device.hostname}{" "}
                        · {application.applicant}
                      </p>
                    </div>
                    <StatusTag status={application.status} t={t} />
                  </div>
                ))}
              </div>
            ) : (
              <div className="mt-4 border-t border-border pt-4 text-xs text-muted-foreground">
                {t("relay.overview.noActivity")}
              </div>
            )}
          </div>
        </aside>
      </section>
    </div>
  )
}
