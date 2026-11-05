import { useEffect, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Activity,
  Bell,
  Check,
  GitBranch,
  Plus,
  RadioTower,
  RefreshCw,
  SlidersHorizontal,
  TriangleAlert,
  Waypoints,
  type LucideIcon,
} from "lucide-react"
import { peerRelayApi } from "@/api/peer-relay"
import { ApiProblemError } from "@/api/http"
import { AppSidebar } from "@/components/app-sidebar"
import { LanguageToggle } from "@/components/language-toggle"
import { ModeToggle } from "@/components/mode-toggle"
import { OidcLoginPage } from "@/features/auth/oidc-login-page"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import {
  ConfirmDialog,
  ApplicationDialog,
  RelayRegistrationDialog,
  ReviewDialog,
} from "@/features/peer-relay/components/dialogs"
import { ApplicationsPage } from "@/features/peer-relay/pages/applications-page"
import { EditorPage } from "@/features/peer-relay/pages/editor-page"
import { OverviewPage } from "@/features/peer-relay/pages/overview-page"
import { PolicyPage } from "@/features/peer-relay/pages/policy-page"
import { RelaysPage } from "@/features/peer-relay/pages/relays-page"
import { SettingsPage } from "@/features/peer-relay/pages/settings-page"
import { errorMessage, relayUpdateInput } from "@/features/peer-relay/utils"
import type {
  AssignmentInput,
  Device,
  PeerRelaySettingsInput,
  PolicyRevision,
  Relay,
  RelayApplication,
  RelayUpdate,
  ReviewInput,
} from "@/api/generated/model"
import { useI18n } from "@/i18n"
import { cn } from "@/lib/utils"

type PageKey =
  "overview" | "editor" | "relays" | "requests" | "policy" | "settings"

const EMPTY_DEVICES: Device[] = []
const EMPTY_RELAY_IDS: string[] = []

const queryKeys = {
  dashboard: ["peer-relay", "dashboard"] as const,
  devices: ["peer-relay", "devices"] as const,
  relays: ["peer-relay", "relays"] as const,
  applications: ["peer-relay", "applications"] as const,
  graph: (nodeId: string) => ["peer-relay", "graph", nodeId] as const,
  revisions: ["peer-relay", "revisions"] as const,
  audits: ["peer-relay", "audits"] as const,
  settings: ["peer-relay", "settings"] as const,
}

export function PeerRelayConsole() {
  const { t, locale } = useI18n()
  const queryClient = useQueryClient()
  const [page, setPage] = useState<PageKey>("overview")
  const [selectedDeviceId, setSelectedDeviceId] = useState("")
  const [search, setSearch] = useState("")
  const [draftSelection, setDraftSelection] = useState<{
    key: string
    relayIds: string[]
  } | null>(null)
  const [toast, setToast] = useState("")
  const [applicationOpen, setApplicationOpen] = useState(false)
  const [reviewTarget, setReviewTarget] = useState<{
    application: RelayApplication
    decision: "approved" | "rejected"
  } | null>(null)
  const [reviewNote, setReviewNote] = useState("")
  const [relayFormOpen, setRelayFormOpen] = useState(false)
  const [relayEditTarget, setRelayEditTarget] = useState<Relay | null>(null)
  const [relayCancelTarget, setRelayCancelTarget] = useState<Relay | null>(null)
  const [rollbackTarget, setRollbackTarget] = useState<PolicyRevision | null>(
    null
  )
  const [revisionDetailId, setRevisionDetailId] = useState("")

  const dashboardQuery = useQuery({
    queryKey: queryKeys.dashboard,
    queryFn: peerRelayApi.dashboard,
    refetchInterval: 60_000,
  })
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
  const activeDeviceId = devices.some(
    (device) => device.nodeId === selectedDeviceId
  )
    ? selectedDeviceId
    : (devices[0]?.nodeId ?? "")
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
  const graphQuery = useQuery({
    queryKey: queryKeys.graph(activeDeviceId),
    queryFn: () => peerRelayApi.graph(activeDeviceId),
    enabled: Boolean(currentUser && activeDeviceId),
  })
  const revisionsQuery = useQuery({
    queryKey: queryKeys.revisions,
    queryFn: peerRelayApi.revisions,
    enabled: Boolean(isAdmin && page === "policy"),
  })
  const revisionDetailQuery = useQuery({
    queryKey: [...queryKeys.revisions, "detail", revisionDetailId],
    queryFn: () => peerRelayApi.revisionDetail(revisionDetailId),
    enabled: Boolean(isAdmin && page === "policy" && revisionDetailId),
  })
  const auditsQuery = useQuery({
    queryKey: queryKeys.audits,
    queryFn: peerRelayApi.audits,
    enabled: Boolean(isAdmin && page === "policy"),
  })
  const settingsQuery = useQuery({
    queryKey: queryKeys.settings,
    queryFn: peerRelayApi.settings,
    enabled: Boolean(isAdmin && page === "settings"),
  })
  const graph = graphQuery.data
  const graphDraftKey = graph
    ? `${graph.device.nodeId}:${graph.basePolicyHash}`
    : ""
  const baselineRelayIds = useMemo(
    () =>
      graph?.relays
        .filter((item) =>
          ["platform", "external", "inherited", "unknown"].includes(item.source)
        )
        .map((item) => item.relayNodeId) ?? EMPTY_RELAY_IDS,
    [graph]
  )
  const draftRelayIds =
    draftSelection?.key === graphDraftKey
      ? draftSelection.relayIds
      : baselineRelayIds
  const updateDraftRelayIds = (
    update: string[] | ((current: string[]) => string[])
  ) => {
    if (!graphDraftKey) return
    setDraftSelection((current) => {
      const relayIds =
        current?.key === graphDraftKey ? current.relayIds : baselineRelayIds
      return {
        key: graphDraftKey,
        relayIds: typeof update === "function" ? update(relayIds) : update,
      }
    })
  }

  useEffect(() => {
    if (!toast) return
    const timeout = window.setTimeout(() => setToast(""), 3200)
    return () => window.clearTimeout(timeout)
  }, [toast])

  const refreshAll = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
      queryClient.invalidateQueries({ queryKey: queryKeys.devices }),
      queryClient.invalidateQueries({ queryKey: queryKeys.relays }),
      queryClient.invalidateQueries({ queryKey: queryKeys.applications }),
      queryClient.invalidateQueries({ queryKey: ["peer-relay", "graph"] }),
    ])
  }

  const syncMutation = useMutation({
    mutationFn: peerRelayApi.sync,
    onSuccess: async () => {
      await refreshAll()
      setToast(t("relay.header.synced"))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: peerRelayApi.logout,
    onSuccess: () => {
      queryClient.clear()
      window.location.assign("/login")
    },
  })

  const saveMutation = useMutation({
    mutationFn: (input: AssignmentInput) => peerRelayApi.saveAssignments(input),
    onSuccess: async (result) => {
      queryClient.setQueryData(
        queryKeys.graph(result.graph.device.nodeId),
        result.graph
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
        queryClient.invalidateQueries({ queryKey: queryKeys.audits }),
      ])
      setToast(t("relay.editor.saved"))
    },
  })

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

  const cancelApplicationMutation = useMutation({
    mutationFn: peerRelayApi.cancelApplication,
    onSuccess: () => refreshAll(),
  })

  const reviewMutation = useMutation({
    mutationFn: ({ id, input }: { id: string; input: ReviewInput }) =>
      peerRelayApi.review(id, input),
    onSuccess: async () => {
      setReviewTarget(null)
      setReviewNote("")
      await refreshAll()
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

  const rollbackMutation = useMutation({
    mutationFn: peerRelayApi.rollback,
    onSuccess: async () => {
      setRollbackTarget(null)
      await refreshAll()
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.revisions }),
        queryClient.invalidateQueries({ queryKey: queryKeys.audits }),
      ])
    },
  })

  const settingsMutation = useMutation({
    mutationFn: (input: PeerRelaySettingsInput) =>
      peerRelayApi.saveSettings(input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.settings }),
        queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
        queryClient.invalidateQueries({ queryKey: queryKeys.devices }),
        queryClient.invalidateQueries({ queryKey: queryKeys.relays }),
      ])
      setToast(t("relay.settings.saved"))
    },
  })

  const filteredDevices = useMemo(() => {
    const normalized = search.trim().toLocaleLowerCase()
    if (!normalized) return devices
    return devices.filter((device) =>
      [
        device.hostname,
        device.givenName,
        device.username,
        device.ipv4,
        device.ipv6,
        ...device.tags,
      ]
        .filter(Boolean)
        .some((value) => value!.toLocaleLowerCase().includes(normalized))
    )
  }, [devices, search])

  const dirty = useMemo(() => {
    if (!graph) return false
    const baseline = graph.relays
      .filter((item) =>
        ["platform", "external", "inherited", "unknown"].includes(item.source)
      )
      .map((item) => item.relayNodeId)
      .sort()
    const draft = [...draftRelayIds].sort()
    return (
      baseline.length !== draft.length ||
      baseline.some((id, index) => id !== draft[index])
    )
  }, [draftRelayIds, graph])

  const userApplications = applicationsQuery.data ?? []
  const pendingApplications = userApplications.filter(
    (item) => item.status === "pending"
  )
  const navItems: {
    key: PageKey
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

  const openEditor = (deviceId?: string) => {
    if (deviceId) setSelectedDeviceId(deviceId)
    setPage("editor")
  }

  const toggleRelay = (relayId: string) => {
    updateDraftRelayIds((current) =>
      current.includes(relayId)
        ? current.filter((item) => item !== relayId)
        : [...current, relayId]
    )
  }

  const saveDraft = () => {
    if (!graph || !activeDeviceId) return
    const removed = graph.relays
      .filter(
        (item) =>
          item.source === "platform" &&
          !draftRelayIds.includes(item.relayNodeId)
      )
      .map((item) => item.relay.displayName)
    const added = graph.relays
      .filter(
        (item) => item.source === "" && draftRelayIds.includes(item.relayNodeId)
      )
      .map((item) => item.relay.displayName)
    const summary = [
      added.length ? `${t("relay.editor.summary")}: +${added.length}` : "",
      removed.length ? `−${removed.length}` : "",
    ]
      .filter(Boolean)
      .join("  ")
    if (summary) setToast(summary)
    saveMutation.mutate({
      sourceNodeId: activeDeviceId,
      relayNodeIds: draftRelayIds,
      basePolicyHash: graph.basePolicyHash,
    })
  }

  const pageTitle =
    navItems.find((item) => item.key === page)?.label ?? t("relay.nav.overview")
  const pageDescription =
    page === "overview"
      ? t("relay.overview.description")
      : page === "editor"
        ? t("relay.editor.description")
        : page === "relays"
          ? t("relay.relays.description")
          : page === "requests"
            ? t("relay.applications.description")
            : page === "policy"
              ? t("relay.policy.description")
              : t("relay.settings.description")

  const pageEyebrow =
    page === "overview"
      ? t("relay.overview.eyebrow")
      : page === "editor"
        ? t("relay.editor.eyebrow")
        : page === "relays"
          ? t("relay.relays.eyebrow")
          : page === "requests"
            ? t("relay.applications.eyebrow")
            : page === "policy"
              ? t("relay.policy.eyebrow")
              : t("relay.settings.eyebrow")
  const pageHeading =
    page === "overview"
      ? t("relay.overview.title")
      : page === "editor"
        ? t("relay.editor.title")
        : page === "relays"
          ? t("relay.relays.title")
          : page === "requests"
            ? t("relay.applications.title")
            : page === "policy"
              ? t("relay.policy.title")
              : t("relay.settings.title")

  const apiError = dashboardQuery.error
  const configError =
    apiError instanceof ApiProblemError && apiError.problem.status === 503

  if (apiError instanceof ApiProblemError && apiError.problem.status === 401) {
    return <OidcLoginPage />
  }

  return (
    <SidebarProvider className="min-h-svh bg-sidebar text-foreground">
      <AppSidebar
        items={navItems}
        activePage={page}
        currentUser={currentUser}
        syncError={dashboard?.syncError}
        isLoggingOut={logoutMutation.isPending}
        onNavigate={setPage}
        onLogout={() => logoutMutation.mutate()}
        variant="inset"
      />
      <SidebarInset className="min-h-svh min-w-0">
        <header className="sticky top-0 z-20 border-b border-border bg-background/90 px-5 backdrop-blur-xl sm:px-8 lg:px-10">
          <div className="mx-auto flex h-[72px] max-w-[1320px] items-center justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
              <SidebarTrigger
                aria-label={t("relay.sidebar.toggle")}
                title={t("relay.sidebar.toggle")}
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

        <div className="mx-auto w-full max-w-[1320px] px-5 py-7 sm:px-8 sm:py-9 lg:px-10">
          <div className="mb-7 flex flex-col justify-between gap-4 sm:mb-8 sm:flex-row sm:items-end">
            <div>
              <p className="text-[10px] font-bold tracking-[0.2em] text-muted-foreground">
                {pageEyebrow}
              </p>
              <h2 className="mt-2 font-heading text-[28px] font-medium tracking-[-0.035em] sm:text-[34px]">
                {pageHeading}
              </h2>
              <p className="mt-2 max-w-[760px] text-sm leading-6 text-muted-foreground">
                {pageDescription}
              </p>
            </div>
            {page === "relays" && canManageRelays ? (
              <Button
                className="flex items-center justify-center gap-2 px-4"
                onPress={() => setRelayFormOpen(true)}
              >
                <Plus size={15} /> {t("relay.relays.add")}
              </Button>
            ) : page === "relays" && canApply ? (
              <Button
                className="flex items-center justify-center gap-2 px-4"
                onPress={() => setApplicationOpen(true)}
              >
                <Plus size={15} /> {t("relay.relays.apply")}
              </Button>
            ) : null}
          </div>

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

          {page === "overview" ? (
            <OverviewPage
              dashboard={dashboard}
              devices={devices}
              relays={relays}
              applications={pendingApplications}
              isAdmin={isAdmin}
              isLoading={dashboardQuery.isLoading}
              locale={locale}
              t={t}
              onOpenEditor={openEditor}
              onOpenApplications={() => setPage("requests")}
              onSync={() => syncMutation.mutate()}
              syncPending={syncMutation.isPending}
            />
          ) : null}

          {page === "editor" ? (
            <EditorPage
              devices={filteredDevices}
              graph={graph}
              selectedDeviceId={activeDeviceId}
              draftRelayIds={draftRelayIds}
              search={search}
              dirty={dirty}
              isLoading={graphQuery.isLoading || devicesQuery.isLoading}
              savePending={saveMutation.isPending}
              saveError={saveMutation.error}
              canApply={canApply}
              t={t}
              onSearch={setSearch}
              onSelectDevice={setSelectedDeviceId}
              onToggleRelay={toggleRelay}
              onSave={saveDraft}
              onDiscard={() => graph && updateDraftRelayIds(baselineRelayIds)}
              onApply={() => setApplicationOpen(true)}
            />
          ) : null}

          {page === "relays" ? (
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
          ) : null}

          {page === "requests" ? (
            <ApplicationsPage
              applications={userApplications}
              devices={devices.filter(
                (device) =>
                  device.username.toLocaleLowerCase() ===
                    currentUser?.username?.toLocaleLowerCase() &&
                  device.tags.length === 0 &&
                  !device.missing
              )}
              isReviewer={isReviewer}
              canApply={canApply}
              isLoading={applicationsQuery.isLoading}
              t={t}
              onCreate={() => setApplicationOpen(true)}
              onCancel={(application) =>
                cancelApplicationMutation.mutate(application.id)
              }
              onReview={(application, decision) => {
                setReviewNote("")
                setReviewTarget({ application, decision })
              }}
            />
          ) : null}

          {page === "policy" && isAdmin ? (
            <PolicyPage
              revisions={revisionsQuery.data ?? []}
              revisionDetail={revisionDetailQuery.data}
              revisionDetailLoading={revisionDetailQuery.isLoading}
              revisionDetailError={revisionDetailQuery.error}
              devices={devices}
              relays={relays}
              audits={auditsQuery.data ?? []}
              isLoading={revisionsQuery.isLoading || auditsQuery.isLoading}
              t={t}
              locale={locale}
              onRollback={setRollbackTarget}
              onViewRevision={(revision) => setRevisionDetailId(revision.id)}
            />
          ) : null}

          {page === "settings" && isAdmin ? (
            <SettingsPage
              key={`${settingsQuery.data?.updatedBy ?? "loading"}:${settingsQuery.data?.updatedAt ?? ""}`}
              settings={settingsQuery.data}
              isLoading={settingsQuery.isLoading}
              error={settingsQuery.error}
              isPending={settingsMutation.isPending}
              saveError={settingsMutation.error}
              locale={locale}
              t={t}
              onSave={(input) => settingsMutation.mutate(input)}
            />
          ) : null}
        </div>
      </SidebarInset>

      {toast ? (
        <div
          role="status"
          className="fixed bottom-5 left-1/2 z-50 flex -translate-x-1/2 items-center gap-2 border border-border bg-primary px-4 py-2.5 text-xs font-semibold text-primary-foreground shadow-xl"
        >
          <Check size={14} /> {toast}
        </div>
      ) : null}

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

      {reviewTarget ? (
        <ReviewDialog
          key={reviewTarget.application.id}
          target={reviewTarget}
          note={reviewNote}
          setNote={setReviewNote}
          t={t}
          isPending={reviewMutation.isPending}
          error={reviewMutation.error}
          onClose={() => setReviewTarget(null)}
          onSubmit={(input) =>
            reviewMutation.mutate({ id: reviewTarget.application.id, input })
          }
        />
      ) : null}

      {rollbackTarget ? (
        <ConfirmDialog
          title={t("relay.policy.confirm")}
          description={rollbackTarget.summary}
          confirmLabel={t("relay.policy.confirm")}
          cancelLabel={t("relay.policy.cancel")}
          isPending={rollbackMutation.isPending}
          error={rollbackMutation.error}
          onCancel={() => setRollbackTarget(null)}
          onConfirm={() => rollbackMutation.mutate(rollbackTarget.id)}
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
    </SidebarProvider>
  )
}
