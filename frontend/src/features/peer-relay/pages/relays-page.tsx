import { useState } from "react"
import {
  Activity,
  CircleHelp,
  Globe2,
  MapPin,
  Network,
  Pencil,
  RadioTower,
  Search,
} from "lucide-react"
import type { Relay } from "@/api/generated/model"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Detail,
  EmptyState,
  LoadingState,
  OptionSelect,
  Pagination,
} from "@/features/peer-relay/components/view-primitives"
import { cn } from "@/lib/utils"

export function RelaysPage({
  relays,
  canManageRelays,
  isLoading,
  t,
  onToggle,
  onEdit,
  onCancel,
}: {
  relays: Relay[]
  canManageRelays: boolean
  isLoading: boolean
  t: (key: string) => string
  onToggle: (relay: Relay) => void
  onEdit: (relay: Relay) => void
  onCancel: (relay: Relay) => void
}) {
  const [search, setSearch] = useState("")
  const [region, setRegion] = useState("all")
  const [page, setPage] = useState(0)
  const regions = [
    ...new Set(relays.map((relay) => relay.region).filter(Boolean)),
  ].sort()
  const normalizedSearch = search.trim().toLocaleLowerCase()
  const filteredRelays = relays.filter((relay) => {
    const matchesSearch = [
      relay.displayName,
      relay.device.givenName,
      relay.device.hostname,
      relay.device.username,
      relay.region,
      relay.publicEndpoint,
      relay.isp,
    ]
      .filter(Boolean)
      .join(" ")
      .toLocaleLowerCase()
      .includes(normalizedSearch)
    return matchesSearch && (region === "all" || relay.region === region)
  })
  const pageSize = 24
  const pageCount = Math.max(1, Math.ceil(filteredRelays.length / pageSize))
  const currentPage = Math.min(page, pageCount - 1)
  const visibleRelays = filteredRelays.slice(
    currentPage * pageSize,
    currentPage * pageSize + pageSize
  )
  return (
    <div>
      <div className="mb-4 flex items-center gap-2 border border-border bg-muted px-4 py-3 text-xs leading-5 text-muted-foreground">
        <CircleHelp size={15} />
        {t("relay.relays.note")}
      </div>
      <div className="mb-4 grid gap-3 sm:grid-cols-[minmax(0,1fr)_220px]">
        <div className="relative">
          <Search
            size={15}
            className="absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            value={search}
            onChange={(event) => {
              setPage(0)
              setSearch(event.target.value)
            }}
            placeholder={t("relay.relays.search")}
            className="pl-9"
          />
        </div>
        <OptionSelect
          value={region}
          onValueChange={(value) => {
            setPage(0)
            setRegion(value)
          }}
          ariaLabel={t("relay.relays.region")}
          options={[
            { value: "all", label: t("relay.relays.allRegions") },
            ...regions.map((value) => ({ value: value!, label: value! })),
          ]}
        />
      </div>
      {isLoading ? (
        <LoadingState t={t} />
      ) : filteredRelays.length === 0 ? (
        <EmptyState
          title={
            relays.length === 0
              ? t("relay.relays.noRelays")
              : t("relay.relays.noSearchResults")
          }
          icon={RadioTower}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
          {visibleRelays.map((relay) => (
            <article
              key={relay.nodeId}
              className="rounded-none border border-border bg-card p-5"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 items-center gap-3">
                  <span className="grid size-11 shrink-0 place-items-center rounded-none bg-muted text-muted-foreground">
                    <RadioTower size={19} />
                  </span>
                  <div className="min-w-0">
                    <h3 className="truncate text-sm font-semibold">
                      {relay.displayName}
                    </h3>
                    <p className="mt-1 truncate text-[10px] text-muted-foreground">
                      {relay.device.givenName || relay.device.hostname}
                    </p>
                  </div>
                </div>
                <Badge variant={relay.selectable ? "secondary" : "outline"}>
                  {relay.selectable
                    ? t("relay.relays.statusOpen")
                    : t("relay.relays.statusPaused")}
                </Badge>
              </div>
              <div className="mt-5 grid grid-cols-2 gap-3 border-t border-border pt-4">
                <Detail
                  icon={MapPin}
                  label={t("relay.relays.region")}
                  value={relay.region || t("relay.common.region")}
                />
                <Detail
                  icon={Globe2}
                  label={t("relay.relays.endpoint")}
                  value={relay.publicEndpoint || "—"}
                />
                <Detail
                  icon={Activity}
                  label={t("relay.relays.port")}
                  value={relay.udpPort ? String(relay.udpPort) : "—"}
                />
                <Detail
                  icon={Network}
                  label={t("relay.relays.isp")}
                  value={relay.isp || "—"}
                />
              </div>
              <div className="mt-4 flex items-center justify-between gap-3 bg-muted px-3 py-2.5">
                <span className="min-w-0 truncate text-xs text-muted-foreground">
                  {relay.device.username} ·{" "}
                  {relay.device.ipv4 || relay.device.ipv6 || "—"}
                </span>
                <span
                  className={cn(
                    "size-1.5 shrink-0",
                    relay.device.online
                      ? "bg-primary"
                      : relay.device.online === false
                        ? "bg-muted-foreground/50"
                        : "bg-muted-foreground"
                  )}
                />
              </div>
              {relay.description ? (
                <p className="mt-3 text-[10px] leading-5 text-muted-foreground">
                  {relay.description}
                </p>
              ) : null}
              {canManageRelays ? (
                <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3">
                  <Button
                    variant="ghost"
                    onPress={() => onEdit(relay)}
                    className="h-auto p-0 text-xs tracking-normal text-muted-foreground normal-case"
                  >
                    <Pencil size={13} />
                    {t("relay.relays.edit")}
                  </Button>
                  <Button
                    variant="ghost"
                    onPress={() => onToggle(relay)}
                    className="h-auto p-0 text-xs tracking-normal text-muted-foreground normal-case"
                  >
                    {relay.selectable
                      ? t("relay.relays.pause")
                      : t("relay.relays.reopen")}
                  </Button>
                  <Button
                    variant="destructive"
                    onPress={() => onCancel(relay)}
                    className="h-auto p-0 text-xs tracking-normal normal-case"
                  >
                    {t("relay.relays.cancelRegistration")}
                  </Button>
                </div>
              ) : null}
            </article>
          ))}
        </div>
      )}
      {!isLoading ? (
        <Pagination
          page={currentPage}
          pageCount={pageCount}
          onPage={setPage}
          t={t}
        />
      ) : null}
    </div>
  )
}
