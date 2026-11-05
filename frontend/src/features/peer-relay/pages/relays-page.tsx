import type { Relay, RelayUpdate } from "@/api/generated/model"
import { peerRelayApi } from "@/api/peer-relay"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  ApplicationDialog,
  ConfirmDialog,
  RelayRegistrationDialog,
} from "@/features/peer-relay/components/dialogs"
import { relayUpdateInput } from "@/features/peer-relay/utils"
import { useI18n } from "@/i18n"
import { cn } from "@/lib/utils"
import { useMutation } from "@tanstack/react-query"
import { Activity, Plus, RadioTower } from "lucide-react"
import { useState } from "react"

import { Input } from "@/components/ui/input"
import {
  PageHeading,
  Toast,
} from "@/features/peer-relay/components/page-heading"
import {
  Detail,
  EmptyState,
  LoadingState,
  OptionSelect,
  Pagination,
} from "@/features/peer-relay/components/view-primitives"
import { queryKeys, usePeerRelayData } from "@/features/peer-relay/queries"
import { useToast } from "@/features/peer-relay/use-toast"
import {
  CircleHelp,
  Globe2,
  MapPin,
  Network,
  Pencil,
  Search,
} from "lucide-react"

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

export function RelaysRoute() {
  const { t } = useI18n()
  const {
    queryClient,
    currentUser,
    canApply,
    canManageRelays,
    devices,
    relaysQuery,
    relays,
    refreshAll,
  } = usePeerRelayData()
  const { toast, setToast } = useToast()
  const [applicationOpen, setApplicationOpen] = useState(false)
  const [relayFormOpen, setRelayFormOpen] = useState(false)
  const [relayEditTarget, setRelayEditTarget] = useState<Relay | null>(null)
  const [relayCancelTarget, setRelayCancelTarget] = useState<Relay | null>(null)
  const applicationMutation = useMutation({
    mutationFn: peerRelayApi.apply,
    onSuccess: async () => {
      setApplicationOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.applications }),
        queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
      ])
      setToast(t("relay.applications.submitted"))
    },
  })

  const relayRegistrationMutation = useMutation({
    mutationFn: peerRelayApi.register,
    onSuccess: async () => {
      setRelayFormOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.relays }),
        queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
      ])
    },
  })

  const relayUpdateMutation = useMutation({
    mutationFn: ({ nodeId, input }: { nodeId: string; input: RelayUpdate }) =>
      peerRelayApi.updateRelay(nodeId, input),
    onSuccess: async () => {
      setRelayEditTarget(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.relays }),
        queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
      ])
    },
  })

  const relayCancelMutation = useMutation({
    mutationFn: peerRelayApi.cancelRelay,
    onSuccess: async () => {
      setRelayCancelTarget(null)
      await refreshAll()
    },
  })

  return (
    <>
      <PageHeading section="relays">
        {canManageRelays ? (
          <Button
            className="flex items-center justify-center gap-2 px-4"
            onPress={() => setRelayFormOpen(true)}
          >
            <Plus size={15} /> {t("relay.relays.add")}
          </Button>
        ) : canApply ? (
          <Button
            className="flex items-center justify-center gap-2 px-4"
            onPress={() => setApplicationOpen(true)}
          >
            <Plus size={15} /> {t("relay.relays.apply")}
          </Button>
        ) : null}
      </PageHeading>
      <RelaysPage
        relays={relays}
        canManageRelays={canManageRelays}
        isLoading={relaysQuery.isLoading}
        t={t}
        onToggle={(relay) =>
          relayUpdateMutation.mutate({
            nodeId: relay.nodeId,
            input: relayUpdateInput(relay, !relay.selectable),
          })
        }
        onEdit={setRelayEditTarget}
        onCancel={setRelayCancelTarget}
      />
      {applicationOpen ? (
        <ApplicationDialog
          devices={devices.filter(
            (device) =>
              device.username.toLocaleLowerCase() ===
                currentUser?.username?.toLocaleLowerCase() &&
              device.tags.length === 0 &&
              !device.missing
          )}
          t={t}
          isPending={applicationMutation.isPending}
          error={applicationMutation.error}
          onClose={() => setApplicationOpen(false)}
          onSubmit={(input) => applicationMutation.mutate(input)}
        />
      ) : null}

      {(relayFormOpen || relayEditTarget) && canManageRelays ? (
        <RelayRegistrationDialog
          key={relayEditTarget?.nodeId ?? "new-relay"}
          devices={
            relayEditTarget
              ? []
              : devices.filter(
                  (device) =>
                    !device.missing &&
                    !relays.some((relay) => relay.nodeId === device.nodeId)
                )
          }
          relay={relayEditTarget ?? undefined}
          t={t}
          isPending={
            relayEditTarget
              ? relayUpdateMutation.isPending
              : relayRegistrationMutation.isPending
          }
          error={
            relayEditTarget
              ? relayUpdateMutation.error
              : relayRegistrationMutation.error
          }
          onClose={() => {
            setRelayFormOpen(false)
            setRelayEditTarget(null)
          }}
          onSubmit={(input) => {
            if ("nodeId" in input) {
              relayRegistrationMutation.mutate(input)
            } else if (relayEditTarget) {
              relayUpdateMutation.mutate({
                nodeId: relayEditTarget.nodeId,
                input,
              })
            }
          }}
        />
      ) : null}

      {relayCancelTarget ? (
        <ConfirmDialog
          title={t("relay.relays.cancelRegistration")}
          description={t("relay.relays.cancelConfirm")}
          confirmLabel={t("relay.relays.cancelRegistration")}
          cancelLabel={t("relay.common.cancel")}
          isPending={relayCancelMutation.isPending}
          error={relayCancelMutation.error}
          onCancel={() => setRelayCancelTarget(null)}
          onConfirm={() => relayCancelMutation.mutate(relayCancelTarget.nodeId)}
        />
      ) : null}

      <Toast message={toast} />
    </>
  )
}
