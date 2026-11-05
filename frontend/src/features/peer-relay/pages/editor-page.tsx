import { useState } from "react"
import {
  Check,
  ChevronRight,
  LockKeyhole,
  Plus,
  RadioTower,
  RefreshCw,
  Search,
  Server,
  SlidersHorizontal,
  TriangleAlert,
} from "lucide-react"
import type { Device, RelayGraph } from "@/api/generated/model"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  DeviceIcon,
  EmptyState,
  LoadingState,
  Notice,
  Pagination,
  SourceLegend,
  SourceTag,
} from "@/features/peer-relay/components/view-primitives"
import { errorMessage } from "@/features/peer-relay/utils"
import { cn } from "@/lib/utils"

export function EditorPage({
  devices,
  graph,
  selectedDeviceId,
  draftRelayIds,
  search,
  dirty,
  isLoading,
  savePending,
  saveError,
  canApply,
  t,
  onSearch,
  onSelectDevice,
  onToggleRelay,
  onSave,
  onDiscard,
  onApply,
}: {
  devices: Device[]
  graph?: RelayGraph
  selectedDeviceId: string
  draftRelayIds: string[]
  search: string
  dirty: boolean
  isLoading: boolean
  savePending: boolean
  saveError: Error | null
  canApply: boolean
  t: (key: string) => string
  onSearch: (value: string) => void
  onSelectDevice: (id: string) => void
  onToggleRelay: (id: string) => void
  onSave: () => void
  onDiscard: () => void
  onApply: () => void
}) {
  const selected = devices.find((device) => device.nodeId === selectedDeviceId)
  const links = graph?.relays ?? []
  const [devicePage, setDevicePage] = useState(0)
  const [relayPage, setRelayPage] = useState(0)
  const filteredDevices = devices.filter((device) =>
    [
      device.givenName,
      device.hostname,
      device.username,
      device.ipv4,
      device.ipv6,
      device.tags.join(" "),
    ]
      .filter(Boolean)
      .join(" ")
      .toLocaleLowerCase()
      .includes(search.trim().toLocaleLowerCase())
  )
  const devicePageCount = Math.max(1, Math.ceil(filteredDevices.length / 40))
  const relayPageCount = Math.max(1, Math.ceil(links.length / 30))
  const currentDevicePage = Math.min(devicePage, devicePageCount - 1)
  const currentRelayPage = Math.min(relayPage, relayPageCount - 1)
  const visibleDevices = filteredDevices.slice(
    currentDevicePage * 40,
    currentDevicePage * 40 + 40
  )
  const visibleLinks = links.slice(
    currentRelayPage * 30,
    currentRelayPage * 30 + 30
  )
  const visibleSelectedLinks = visibleLinks.filter((item) =>
    draftRelayIds.includes(item.relayNodeId)
  )
  const activeCount = links.filter((item) =>
    ["platform", "external", "inherited", "unknown"].includes(item.source)
  ).length
  const newCount = draftRelayIds.filter(
    (id) => !links.some((item) => item.relayNodeId === id && item.source)
  ).length
  const removedCount = links.filter(
    (item) =>
      item.source === "platform" && !draftRelayIds.includes(item.relayNodeId)
  ).length
  const isWritable = Boolean(graph?.canEdit && graph.policyWritesEnabled)

  return (
    <div className="overflow-hidden rounded-none border border-border bg-card">
      <div className="flex flex-col justify-between gap-4 border-b border-border px-4 py-4 sm:flex-row sm:items-center sm:px-6">
        <div className="relative w-full sm:max-w-[280px]">
          <Search
            size={15}
            className="absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            value={search}
            onChange={(event) => {
              setDevicePage(0)
              onSearch(event.target.value)
            }}
            placeholder={t("relay.editor.search")}
            className="pl-9"
          />
        </div>
        <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
          <SourceLegend source="platform" t={t} />
          <SourceLegend source="external" t={t} />
          <SourceLegend source="inherited" t={t} />
          <SourceLegend source="unknown" t={t} />
        </div>
      </div>

      <div className="grid min-h-[490px] lg:grid-cols-[255px_minmax(0,1fr)]">
        <aside className="border-b border-border p-4 sm:p-5 lg:border-r lg:border-b-0">
          <div className="mb-3 flex items-center justify-between px-1">
            <p className="text-[10px] font-bold tracking-[0.15em] text-muted-foreground">
              {t("relay.editor.sourceDevice")}
            </p>
            <SlidersHorizontal size={14} className="text-muted-foreground" />
          </div>
          <div className="space-y-1.5">
            {visibleDevices.map((device) => {
              const active = selectedDeviceId === device.nodeId
              return (
                <Button
                  key={device.nodeId}
                  variant="ghost"
                  onPress={() => onSelectDevice(device.nodeId)}
                  className={cn(
                    "flex w-full items-center gap-3 px-2.5 py-2.5 text-left tracking-normal normal-case",
                    active
                      ? "bg-accent text-accent-foreground"
                      : "hover:bg-muted"
                  )}
                >
                  <DeviceIcon tags={device.tags} small />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs font-semibold">
                      {device.givenName || device.hostname}
                    </span>
                    <span className="mt-1 block truncate font-mono text-[9px] text-muted-foreground">
                      {device.ipv4 || device.ipv6 || device.hostname}
                    </span>
                  </span>
                  {active ? (
                    <ChevronRight size={13} className="text-muted-foreground" />
                  ) : null}
                </Button>
              )
            })}
            {filteredDevices.length === 0 && !isLoading ? (
              <p className="px-2 py-5 text-xs leading-5 text-muted-foreground">
                {t("relay.editor.noDevice")}
              </p>
            ) : null}
          </div>
          <Pagination
            page={currentDevicePage}
            pageCount={devicePageCount}
            onPage={setDevicePage}
            t={t}
          />
        </aside>

        <section className="min-w-0 p-4 sm:p-6">
          {selected ? (
            <>
              <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
                <div className="flex min-w-0 items-center gap-3">
                  <DeviceIcon tags={selected.tags} />
                  <div className="min-w-0">
                    <h3 className="truncate text-sm font-semibold">
                      {selected.givenName || selected.hostname}
                    </h3>
                    <p className="mt-1 font-mono text-[10px] text-muted-foreground">
                      {selected.ipv4 || selected.ipv6 || selected.hostname}{" "}
                      <span className="mx-1.5">·</span>
                      {selected.username}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="rounded-full bg-muted px-2.5 py-1 text-[10px] font-medium text-muted-foreground">
                    {activeCount} {t("relay.overview.assignments")}
                  </span>
                  {dirty ? (
                    <span className="rounded-full bg-muted px-2.5 py-1 text-[10px] font-semibold text-muted-foreground">
                      {t("relay.editor.summary")}
                    </span>
                  ) : null}
                </div>
              </div>

              {graph?.policyStale ? (
                <Notice
                  icon={TriangleAlert}
                  tone="amber"
                  text={t("relay.editor.policyUnavailable")}
                />
              ) : null}
              {!graph?.policyWritesEnabled ? (
                <Notice
                  icon={LockKeyhole}
                  tone="neutral"
                  text={t("relay.editor.writesDisabled")}
                />
              ) : null}
              {graph?.policyWritesEnabled &&
              !graph.canEdit &&
              !graph.policyStale &&
              selected.tags.length === 0 ? (
                <Notice
                  icon={LockKeyhole}
                  tone="neutral"
                  text={t("relay.editor.selfServiceDisabled")}
                />
              ) : null}
              {saveError ? (
                <Notice
                  icon={TriangleAlert}
                  tone="red"
                  text={errorMessage(saveError, t("errors.unknown"))}
                />
              ) : null}

              <div className="mt-6 grid gap-4 xl:grid-cols-[minmax(180px,.7fr)_minmax(0,1.3fr)]">
                <div className="relative overflow-hidden rounded-none border border-border bg-muted p-5">
                  <div className="absolute -top-16 -right-16 size-44 rounded-full border border-border" />
                  <div className="absolute -top-9 -right-9 size-28 rounded-full border border-border" />
                  <p className="relative text-[9px] font-bold tracking-[0.16em] text-muted-foreground">
                    {t("relay.editor.sourceDevice")}
                  </p>
                  <div className="relative mt-6 flex items-center gap-3">
                    <DeviceIcon tags={selected.tags} />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold">
                        {selected.givenName || selected.hostname}
                      </p>
                      <p className="mt-1 truncate font-mono text-[9px] text-muted-foreground">
                        {selected.ipv4 || selected.ipv6 || "—"}
                      </p>
                    </div>
                  </div>
                  <div className="relative mt-6 flex items-center gap-2 text-xs text-muted-foreground">
                    <span
                      className={cn(
                        "size-1.5",
                        selected.online
                          ? "bg-primary"
                          : selected.online === false
                            ? "bg-muted-foreground/50"
                            : "bg-muted-foreground"
                      )}
                    />
                    {selected.online
                      ? t("relay.common.online")
                      : selected.online === false
                        ? t("relay.common.offline")
                        : t("relay.common.unknown")}
                  </div>
                </div>

                <div className="min-w-0">
                  <div className="mb-4 border border-border bg-muted/30 p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <p className="text-[10px] font-bold tracking-[0.16em] text-muted-foreground">
                        {t("relay.editor.topology")}
                      </p>
                      <span className="text-[10px] text-muted-foreground">
                        {draftRelayIds.length}
                      </span>
                    </div>
                    {visibleSelectedLinks.length === 0 ? (
                      <p className="py-3 text-center text-[10px] text-muted-foreground">
                        {t("relay.editor.noEdges")}
                      </p>
                    ) : (
                      <div className="space-y-2">
                        {visibleSelectedLinks.map((item) => (
                          <div
                            key={item.relayNodeId}
                            className="grid grid-cols-[minmax(0,1fr)_44px_minmax(0,1fr)] items-center gap-2"
                          >
                            <div className="min-w-0 border border-border bg-card px-2.5 py-2">
                              <p className="truncate text-[10px] font-semibold">
                                {selected.givenName || selected.hostname}
                              </p>
                              <p className="mt-1 truncate font-mono text-[9px] text-muted-foreground">
                                {selected.ipv4 || selected.ipv6 || "—"}
                              </p>
                            </div>
                            <svg
                              viewBox="0 0 44 14"
                              role="img"
                              aria-label={`${selected.hostname} → ${item.relay.displayName}`}
                              className={cn(
                                "h-4 w-full",
                                item.source === "platform"
                                  ? "text-primary"
                                  : "text-muted-foreground"
                              )}
                            >
                              <circle
                                cx="3"
                                cy="7"
                                r="2.5"
                                fill="currentColor"
                              />
                              <path
                                d="M6 7H37"
                                stroke="currentColor"
                                strokeWidth="1.5"
                              />
                              <path
                                d="m33 3 5 4-5 4"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="1.5"
                              />
                            </svg>
                            <div className="min-w-0 border border-border bg-card px-2.5 py-2">
                              <p className="truncate text-[10px] font-semibold">
                                {item.relay.displayName}
                              </p>
                              <p className="mt-1 truncate text-[9px] text-muted-foreground">
                                {item.relay.region ||
                                  item.relay.device.givenName ||
                                  item.relay.device.hostname}
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="mb-3 flex items-center justify-between px-1">
                    <p className="text-[10px] font-bold tracking-[0.16em] text-muted-foreground">
                      {t("relay.editor.available")}
                    </p>
                    <span className="text-[10px] text-muted-foreground">
                      {t("relay.editor.selected")}: {draftRelayIds.length}
                    </span>
                  </div>
                  <div className="space-y-2">
                    {visibleLinks.map((item) => {
                      const checked = draftRelayIds.includes(item.relayNodeId)
                      const immutable = [
                        "external",
                        "inherited",
                        "unknown",
                      ].includes(item.source)
                      const disabled =
                        !isWritable ||
                        immutable ||
                        (!item.relay.selectable && !checked)
                      return (
                        <div
                          key={item.relayNodeId}
                          className={cn(
                            "flex items-center gap-3 border px-3.5 py-3 transition",
                            checked
                              ? "border-primary bg-accent"
                              : "border-border"
                          )}
                        >
                          <div
                            className={cn(
                              "relative flex w-8 shrink-0 items-center justify-center",
                              checked
                                ? "text-primary"
                                : "text-muted-foreground/50"
                            )}
                          >
                            <div className="h-px w-7 bg-current" />
                            <div className="absolute right-0 grid size-3 place-items-center rounded-full border border-current bg-card">
                              {checked ? (
                                <span className="size-1 rounded-full bg-current" />
                              ) : null}
                            </div>
                          </div>
                          <Button
                            variant="ghost"
                            isDisabled={disabled}
                            className="flex min-w-0 flex-1 items-center gap-3 text-left tracking-normal normal-case"
                            onPress={() =>
                              !disabled && onToggleRelay(item.relayNodeId)
                            }
                          >
                            <span className="grid size-9 shrink-0 place-items-center rounded-none bg-muted text-muted-foreground">
                              <RadioTower size={16} />
                            </span>
                            <span className="min-w-0 flex-1">
                              <span className="flex items-center gap-2">
                                <span className="truncate text-xs font-semibold">
                                  {item.relay.displayName}
                                </span>
                                <SourceTag source={item.source} t={t} />
                              </span>
                              <span className="mt-1.5 flex min-w-0 items-center gap-2 text-[9px] text-muted-foreground">
                                <span className="truncate">
                                  {item.relay.region ||
                                    t("relay.common.region")}
                                </span>
                                <span>·</span>
                                <span className="truncate">
                                  {item.relay.device.givenName ||
                                    item.relay.device.hostname}
                                </span>
                              </span>
                              {item.warning ? (
                                <span className="mt-1 block text-[9px] text-muted-foreground">
                                  {item.warning}
                                </span>
                              ) : null}
                            </span>
                          </Button>
                          <Button
                            variant="outline"
                            size="icon-xs"
                            onPress={() =>
                              !disabled && onToggleRelay(item.relayNodeId)
                            }
                            isDisabled={disabled}
                            aria-label={item.relay.displayName}
                            aria-pressed={checked}
                            className={cn(
                              "size-5 shrink-0 p-0 transition",
                              checked &&
                                "border-primary bg-primary text-primary-foreground",
                              disabled && "cursor-not-allowed opacity-70"
                            )}
                          >
                            {checked ? (
                              <Check size={13} strokeWidth={2.8} />
                            ) : null}
                          </Button>
                        </div>
                      )
                    })}
                    {links.length === 0 ? (
                      <div className="rounded-none border border-dashed border-border px-4 py-8 text-center">
                        <RadioTower
                          size={20}
                          className="mx-auto text-muted-foreground"
                        />
                        <p className="mt-3 text-xs text-muted-foreground">
                          {t("relay.editor.noRelays")}
                        </p>
                      </div>
                    ) : null}
                  </div>
                  <Pagination
                    page={currentRelayPage}
                    pageCount={relayPageCount}
                    onPage={setRelayPage}
                    t={t}
                  />
                  {canApply ? (
                    <Button
                      variant="outline"
                      className="mt-3 flex w-full items-center justify-center gap-2 border-dashed text-xs tracking-normal text-muted-foreground normal-case"
                      onPress={onApply}
                    >
                      <Plus size={13} />
                      {t("relay.relays.apply")}
                    </Button>
                  ) : null}
                </div>
              </div>

              <div className="mt-6 flex flex-col gap-3 border-t border-border pt-4 sm:flex-row sm:items-center sm:justify-between">
                <p className="max-w-xl text-[10px] leading-5 text-muted-foreground">
                  {t("relay.editor.routeNotice")}
                </p>
                <div className="flex items-center gap-2">
                  {dirty ? (
                    <Button
                      variant="ghost"
                      onPress={onDiscard}
                      className="px-3 text-xs tracking-normal text-muted-foreground normal-case"
                    >
                      {t("relay.editor.discard")}
                    </Button>
                  ) : null}
                  <Button
                    isDisabled={
                      !dirty ||
                      !isWritable ||
                      savePending ||
                      !graph?.basePolicyHash
                    }
                    onPress={onSave}
                    className="flex items-center justify-center gap-2"
                  >
                    {savePending ? (
                      <RefreshCw size={13} className="animate-spin" />
                    ) : (
                      <Check size={14} />
                    )}
                    {savePending
                      ? t("relay.editor.saving")
                      : `${t("relay.editor.save")} ${newCount || removedCount ? `· +${newCount} / −${removedCount}` : ""}`}
                  </Button>
                </div>
              </div>
            </>
          ) : isLoading ? (
            <LoadingState t={t} />
          ) : (
            <EmptyState title={t("relay.editor.noDevice")} icon={Server} />
          )}
        </section>
      </div>
    </div>
  )
}
