import type {
  Device,
  ManagedPolicySelectorRule,
  PolicyRevision,
  PolicySelectorRuleInput,
  PolicySelectorRulePreview,
  Relay,
} from "@/api/generated/model"
import { peerRelayApi } from "@/api/peer-relay"
import { Button } from "@/components/ui/button"
import { ConfirmDialog } from "@/features/peer-relay/components/dialogs"
import { useI18n } from "@/i18n"
import { cn } from "@/lib/utils"
import { useMutation, useQuery } from "@tanstack/react-query"
import { Activity, GitBranch, Plus } from "lucide-react"
import { useState } from "react"

import type {
  PolicyRevisionDetail,
  PolicySelectorRulePreviewInput,
} from "@/api/generated/model"
import { Input } from "@/components/ui/input"
import {
  PageHeading,
  Toast,
} from "@/features/peer-relay/components/page-heading"
import {
  EmptyState,
  HashChip,
  LoadingState,
} from "@/features/peer-relay/components/view-primitives"
import { queryKeys, usePeerRelayData } from "@/features/peer-relay/queries"
import { useToast } from "@/features/peer-relay/use-toast"
import { formatDate } from "@/features/peer-relay/utils"
import { Clock3, Trash2 } from "lucide-react"

export function PolicyPage({
  revisions,
  revisionDetail,
  revisionDetailLoading,
  revisionDetailError,
  devices,
  relays,
  audits,
  selectorRules,
  selectorRulesLoading,
  selectorRulesError,
  selectorPreview,
  selectorPreviewPending,
  selectorMutationPending,
  selectorMutationError,
  policyWritesEnabled,
  isLoading,
  t,
  locale,
  onRollback,
  onViewRevision,
  onPreviewSelectorRule,
  onCreateSelectorRule,
  onDeleteSelectorRule,
}: {
  revisions: PolicyRevision[]
  revisionDetail?: PolicyRevisionDetail
  revisionDetailLoading: boolean
  revisionDetailError?: Error | null
  devices: Device[]
  relays: Relay[]
  audits: {
    id: number
    actor: string
    action: string
    target: string
    before?: string
    after?: string
    result: string
    createdAt: string
  }[]
  selectorRules: ManagedPolicySelectorRule[]
  selectorRulesLoading: boolean
  selectorRulesError?: Error | null
  selectorPreview?: PolicySelectorRulePreview
  selectorPreviewPending: boolean
  selectorMutationPending: boolean
  selectorMutationError?: Error | null
  policyWritesEnabled: boolean
  isLoading: boolean
  t: (key: string) => string
  locale: string
  onRollback: (revision: PolicyRevision) => void
  onViewRevision: (revision: PolicyRevision) => void
  onPreviewSelectorRule: (input: PolicySelectorRulePreviewInput) => void
  onCreateSelectorRule: (input: PolicySelectorRuleInput) => void
  onDeleteSelectorRule: (rule: ManagedPolicySelectorRule) => void
}) {
  const [sourceSelector, setSourceSelector] = useState("")
  const [relayNodeId, setRelayNodeId] = useState("")
  const previewIsCurrent =
    selectorPreview?.sourceSelector === sourceSelector.trim() &&
    selectorPreview?.relayNodeId === relayNodeId
  const deviceLabel = (selector: string) => {
    const device = devices.find(
      (item) => item.ipv4 === selector || item.ipv6 === selector
    )
    return device ? device.givenName || device.hostname : selector
  }
  const relayLabel = (selector: string) => {
    const relay = relays.find(
      (item) => item.device.ipv4 === selector || item.device.ipv6 === selector
    )
    return relay ? relay.displayName : selector
  }

  return (
    <div className="space-y-5">
      <section className="border border-border bg-card">
        <div className="border-b border-border px-5 py-4">
          <h3 className="text-sm font-semibold">
            {t("relay.policy.selectorRules")}
          </h3>
          <p className="mt-2 max-w-3xl text-xs leading-5 text-muted-foreground">
            {t("relay.policy.selectorHelp")}
          </p>
        </div>
        <div className="grid gap-5 p-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
          <div className="space-y-4">
            <label
              className="block text-[10px] font-bold tracking-widest text-muted-foreground"
              htmlFor="policy-selector-source"
            >
              {t("relay.policy.sourceSelector")}
              <Input
                id="policy-selector-source"
                value={sourceSelector}
                onChange={(event) => setSourceSelector(event.target.value)}
                placeholder="chichu@ 或 group:admins"
                autoComplete="off"
                className="mt-2"
              />
            </label>
            <label
              className="block text-[10px] font-bold tracking-widest text-muted-foreground"
              htmlFor="policy-selector-relay"
            >
              {t("relay.policy.targetRelay")}
              <select
                id="policy-selector-relay"
                value={relayNodeId}
                onChange={(event) => setRelayNodeId(event.target.value)}
                className="mt-2 h-10 w-full border border-border bg-background px-3 text-xs text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/30"
              >
                <option value="">{t("relay.policy.chooseRelay")}</option>
                {relays
                  .filter(
                    (relay) =>
                      relay.selectable &&
                      !relay.device.missing &&
                      Boolean(relay.device.ipv4)
                  )
                  .map((relay) => (
                    <option key={relay.nodeId} value={relay.nodeId}>
                      {relay.displayName} · {relay.device.ipv4}
                    </option>
                  ))}
              </select>
            </label>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                isDisabled={
                  !sourceSelector.trim() ||
                  !relayNodeId ||
                  selectorPreviewPending ||
                  selectorMutationPending
                }
                onPress={() =>
                  onPreviewSelectorRule({
                    sourceSelector: sourceSelector.trim(),
                    relayNodeId,
                  })
                }
              >
                <Activity />
                {selectorPreviewPending
                  ? t("relay.policy.previewingScope")
                  : t("relay.policy.previewScope")}
              </Button>
              {previewIsCurrent && selectorPreview ? (
                <Button
                  size="sm"
                  isDisabled={!policyWritesEnabled || selectorMutationPending}
                  onPress={() =>
                    onCreateSelectorRule({
                      sourceSelector: selectorPreview.sourceSelector,
                      relayNodeId: selectorPreview.relayNodeId,
                      basePolicyHash: selectorPreview.basePolicyHash,
                      scopeHash: selectorPreview.scopeHash,
                    })
                  }
                >
                  <Plus />
                  {selectorMutationPending
                    ? t("relay.policy.savingSelectorRule")
                    : t("relay.policy.createSelectorRule")}
                </Button>
              ) : null}
            </div>
            {!policyWritesEnabled ? (
              <p className="text-[10px] text-muted-foreground">
                {t("relay.editor.writesDisabled")}
              </p>
            ) : null}
            {selectorPreviewPending && !selectorPreview ? (
              <LoadingState t={t} />
            ) : null}
            {selectorPreview && previewIsCurrent ? (
              <div className="border border-border p-3">
                <p className="text-xs font-semibold">
                  {selectorPreview.sourceSelector} →{" "}
                  {selectorPreview.relay.displayName}
                </p>
                <p className="mt-1 text-[10px] text-muted-foreground">
                  {selectorPreview.relaySelector} ·{" "}
                  {t("relay.policy.currentMatches")}
                  {selectorPreview.sourceDevices.length}
                </p>
                <p className="mt-2 text-[10px] leading-4 text-muted-foreground">
                  {selectorPreview.warning}
                </p>
                {selectorPreview.sourceDevices.length ? (
                  <ul className="mt-2 max-h-36 space-y-1 overflow-auto text-[10px] text-muted-foreground">
                    {selectorPreview.sourceDevices.map((device) => (
                      <li key={device.nodeId} className="truncate">
                        {device.givenName || device.hostname} ·{" "}
                        {device.username} · {device.ipv4 || device.ipv6}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            ) : null}
            {selectorMutationError ? (
              <p role="alert" className="text-xs text-destructive">
                {selectorMutationError.message || t("errors.unknown")}
              </p>
            ) : null}
          </div>
          <div className="space-y-3">
            {selectorRulesError ? (
              <p role="alert" className="text-xs text-destructive">
                {selectorRulesError.message || t("errors.unknown")}
              </p>
            ) : selectorRulesLoading ? (
              <LoadingState t={t} />
            ) : selectorRules.length === 0 ? (
              <EmptyState
                title={t("relay.policy.noSelectorRules")}
                icon={GitBranch}
              />
            ) : (
              selectorRules.map((rule) => (
                <div key={rule.id} className="border border-border p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-xs font-semibold">
                        {rule.sourceSelector} →{" "}
                        {rule.relay?.displayName ?? rule.relaySelector}
                      </p>
                      <p className="mt-1 text-[10px] text-muted-foreground">
                        {rule.relaySelector} ·{" "}
                        {t("relay.policy.currentMatches")}
                        {rule.sourceDevices.length}
                        {rule.targetChanged
                          ? ` · ${t("relay.policy.targetChanged")}`
                          : ""}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`${t("relay.policy.removeSelectorRule")}: ${rule.sourceSelector}`}
                      isDisabled={!rule.canDelete || selectorMutationPending}
                      onPress={() => onDeleteSelectorRule(rule)}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                  {rule.warning ? (
                    <p className="mt-2 text-[10px] leading-4 text-muted-foreground">
                      {rule.warning}
                    </p>
                  ) : null}
                  {rule.sourceDevices.length ? (
                    <p className="mt-2 truncate text-[10px] text-muted-foreground">
                      {rule.sourceDevices
                        .map((device) => device.givenName || device.hostname)
                        .join(" · ")}
                    </p>
                  ) : null}
                </div>
              ))
            )}
          </div>
        </div>
      </section>
      {revisionDetail || revisionDetailLoading || revisionDetailError ? (
        <section className="rounded-none border border-border bg-card">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <div>
              <p className="text-[10px] font-bold tracking-[0.15em] text-muted-foreground">
                {t("relay.policy.revisionDiff")}
              </p>
              <h3 className="mt-1 text-sm font-semibold">
                {revisionDetail?.summary ?? t("relay.common.loading")}
              </h3>
            </div>
          </div>
          {revisionDetailLoading ? (
            <LoadingState t={t} />
          ) : revisionDetailError ? (
            <p className="p-5 text-xs text-destructive">
              {revisionDetailError.message || t("errors.unknown")}
            </p>
          ) : revisionDetail ? (
            <div className="space-y-4 p-5">
              {revisionDetail.status !== "success" ? (
                <p className="border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground">
                  {t("relay.policy.notApplied")}
                </p>
              ) : null}
              {revisionDetail.otherPolicyChanged ? (
                <p className="border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground">
                  {t("relay.policy.otherPolicyChanged")}
                </p>
              ) : null}
              <div className="grid gap-4 md:grid-cols-2">
                {[
                  {
                    title: t("relay.policy.addedRelayGrants"),
                    grants: revisionDetail.addedRelayGrants,
                    tone: "text-primary",
                  },
                  {
                    title: t("relay.policy.removedRelayGrants"),
                    grants: revisionDetail.removedRelayGrants,
                    tone: "text-destructive",
                  },
                ].map(({ title, grants, tone }) => (
                  <div key={title} className="border border-border p-3">
                    <h4 className={`text-xs font-semibold ${tone}`}>
                      {title} · {grants.length}
                    </h4>
                    {grants.length === 0 ? (
                      <p className="mt-2 text-[10px] text-muted-foreground">
                        {t("relay.common.noData")}
                      </p>
                    ) : (
                      <ul className="mt-2 space-y-2">
                        {grants.map((grant, index) => (
                          <li
                            key={`${grant.src.join(",")}-${grant.dst.join(",")}-${index}`}
                            className="text-[10px] leading-5 text-muted-foreground"
                          >
                            <span className="font-medium text-foreground">
                              {t("relay.policy.affectedDevices")}:
                            </span>
                            {grant.src.map(deviceLabel).join(", ")}
                            <span className="mx-2">→</span>
                            <span className="font-medium text-foreground">
                              {t("relay.policy.affectedRelays")}:
                            </span>
                            {grant.dst.map(relayLabel).join(", ")}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
              <details>
                <summary className="cursor-pointer text-xs text-muted-foreground">
                  {t("relay.policy.fullSnapshots")}
                </summary>
                <div className="mt-3 grid gap-3 xl:grid-cols-2">
                  <div className="min-w-0">
                    <p className="mb-1 text-[10px] font-semibold text-muted-foreground">
                      {t("relay.policy.beforePolicy")}
                    </p>
                    <pre className="max-h-80 overflow-auto border border-border bg-muted/40 p-3 text-[10px] leading-5 break-all whitespace-pre-wrap">
                      {revisionDetail.before}
                    </pre>
                  </div>
                  <div className="min-w-0">
                    <p className="mb-1 text-[10px] font-semibold text-muted-foreground">
                      {t("relay.policy.afterPolicy")}
                    </p>
                    <pre className="max-h-80 overflow-auto border border-border bg-muted/40 p-3 text-[10px] leading-5 break-all whitespace-pre-wrap">
                      {revisionDetail.after}
                    </pre>
                  </div>
                </div>
              </details>
            </div>
          ) : null}
        </section>
      ) : null}

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,.8fr)]">
        <section className="rounded-none border border-border bg-card">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <div>
              <h3 className="text-sm font-semibold">
                {t("relay.policy.revisions")}
              </h3>
            </div>
            <span className="rounded-full bg-muted px-2.5 py-1 text-[10px] text-muted-foreground">
              {revisions.length}
            </span>
          </div>
          <div className="divide-y divide-border">
            {isLoading ? (
              <LoadingState t={t} />
            ) : revisions.length === 0 ? (
              <div className="p-5">
                <EmptyState
                  title={t("relay.policy.noHistory")}
                  icon={GitBranch}
                />
              </div>
            ) : (
              revisions.map((revision) => (
                <article key={revision.id} className="p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span
                          className={cn(
                            "size-2",
                            revision.status === "success"
                              ? "bg-primary"
                              : "bg-destructive"
                          )}
                        />
                        <h4 className="text-xs font-semibold">
                          {revision.summary}
                        </h4>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                        <span>{revision.actor}</span>
                        <span>{formatDate(revision.createdAt, locale)}</span>
                        <span>{t(`relay.policy.${revision.status}`)}</span>
                      </div>
                    </div>
                    <div className="flex shrink-0 gap-2">
                      <Button
                        variant="outline"
                        onPress={() => onViewRevision(revision)}
                        className="px-3"
                      >
                        {t("relay.policy.preview")}
                      </Button>
                      {revision.status === "success" ? (
                        <Button
                          variant="outline"
                          isDisabled={
                            revisionDetail?.id !== revision.id ||
                            revisionDetailLoading
                          }
                          onPress={() => onRollback(revision)}
                          className="px-3"
                        >
                          {t("relay.policy.rollback")}
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  <div className="mt-4 grid gap-2 sm:grid-cols-2">
                    <HashChip
                      label={t("relay.policy.before")}
                      value={revision.beforeHash}
                    />
                    <HashChip
                      label={t("relay.policy.after")}
                      value={revision.afterHash}
                    />
                  </div>
                </article>
              ))
            )}
          </div>
        </section>

        <section className="rounded-none border border-border bg-card">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <div>
              <p className="text-[10px] font-bold tracking-[0.15em] text-muted-foreground">
                {t("relay.common.activityLog")}
              </p>
              <h3 className="mt-1 text-sm font-semibold">
                {t("relay.policy.audit")}
              </h3>
            </div>
            <span className="rounded-full bg-muted px-2.5 py-1 text-[10px] text-muted-foreground">
              {audits.length}
            </span>
          </div>
          {isLoading ? (
            <LoadingState t={t} />
          ) : audits.length === 0 ? (
            <div className="p-5">
              <EmptyState title={t("relay.policy.noAudit")} icon={Clock3} />
            </div>
          ) : (
            <div className="max-h-[660px] divide-y divide-border overflow-auto">
              {audits.map((audit) => (
                <div key={audit.id} className="flex gap-3 px-5 py-4">
                  <div className="mt-0.5 grid size-7 shrink-0 place-items-center rounded-none bg-muted text-muted-foreground">
                    <Activity size={13} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium">
                      {audit.action}
                    </p>
                    <p className="mt-1 truncate text-[10px] text-muted-foreground">
                      {audit.actor} · {audit.target} · {audit.result}
                    </p>
                    {audit.before || audit.after ? (
                      <details className="mt-2 text-[10px] text-muted-foreground">
                        <summary className="cursor-pointer select-none">
                          {t("relay.policy.auditDetails")}
                        </summary>
                        <div className="mt-2 grid gap-2">
                          {audit.before ? (
                            <pre className="max-h-36 overflow-auto border border-border bg-muted/40 p-2 break-all whitespace-pre-wrap">
                              {`${t("relay.policy.before")}\n${audit.before}`}
                            </pre>
                          ) : null}
                          {audit.after ? (
                            <pre className="max-h-36 overflow-auto border border-border bg-muted/40 p-2 break-all whitespace-pre-wrap">
                              {`${t("relay.policy.after")}\n${audit.after}`}
                            </pre>
                          ) : null}
                        </div>
                      </details>
                    ) : null}
                  </div>
                  <time className="shrink-0 text-[9px] text-muted-foreground">
                    {formatDate(audit.createdAt, locale)}
                  </time>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

export function PolicyRoute() {
  const { t, locale } = useI18n()
  const { queryClient, dashboard, isAdmin, devices, relays, refreshAll } =
    usePeerRelayData()
  const { toast, setToast } = useToast()
  const [rollbackTarget, setRollbackTarget] = useState<PolicyRevision | null>(
    null
  )
  const [revisionDetailId, setRevisionDetailId] = useState("")
  const [selectorRulePreview, setSelectorRulePreview] =
    useState<PolicySelectorRulePreview>()
  const [selectorRuleDeleteTarget, setSelectorRuleDeleteTarget] =
    useState<ManagedPolicySelectorRule | null>(null)

  const revisionsQuery = useQuery({
    queryKey: queryKeys.revisions,
    queryFn: peerRelayApi.revisions,
    enabled: Boolean(isAdmin),
  })
  const selectorRulesQuery = useQuery({
    queryKey: queryKeys.selectorRules,
    queryFn: peerRelayApi.selectorRules,
    enabled: Boolean(isAdmin),
  })
  const revisionDetailQuery = useQuery({
    queryKey: [...queryKeys.revisions, "detail", revisionDetailId],
    queryFn: () => peerRelayApi.revisionDetail(revisionDetailId),
    enabled: Boolean(isAdmin && revisionDetailId),
  })
  const auditsQuery = useQuery({
    queryKey: queryKeys.audits,
    queryFn: peerRelayApi.audits,
    enabled: Boolean(isAdmin),
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

  const selectorRulePreviewMutation = useMutation({
    mutationFn: peerRelayApi.previewSelectorRule,
    onSuccess: (result) => setSelectorRulePreview(result),
  })

  const refreshPolicyManagement = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.selectorRules }),
      queryClient.invalidateQueries({ queryKey: queryKeys.revisions }),
      queryClient.invalidateQueries({ queryKey: queryKeys.audits }),
      queryClient.invalidateQueries({ queryKey: queryKeys.canvasGraph }),
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
    ])
  }

  const createSelectorRuleMutation = useMutation({
    mutationFn: (input: PolicySelectorRuleInput) =>
      peerRelayApi.createSelectorRule(input),
    onSuccess: async () => {
      setSelectorRulePreview(undefined)
      await refreshPolicyManagement()
      setToast(t("relay.policy.selectorRuleSaved"))
    },
  })

  const deleteSelectorRuleMutation = useMutation({
    mutationFn: ({
      id,
      basePolicyHash,
    }: {
      id: string
      basePolicyHash: string
    }) => peerRelayApi.deleteSelectorRule(id, basePolicyHash),
    onSuccess: async () => {
      setSelectorRuleDeleteTarget(null)
      await refreshPolicyManagement()
      setToast(t("relay.policy.selectorRuleRemoved"))
    },
  })

  return (
    <>
      <PageHeading section="policy" />
      <PolicyPage
        revisions={revisionsQuery.data ?? []}
        revisionDetail={revisionDetailQuery.data}
        revisionDetailLoading={revisionDetailQuery.isLoading}
        revisionDetailError={revisionDetailQuery.error}
        devices={devices}
        relays={relays}
        audits={auditsQuery.data ?? []}
        selectorRules={selectorRulesQuery.data ?? []}
        selectorRulesLoading={selectorRulesQuery.isLoading}
        selectorRulesError={selectorRulesQuery.error}
        selectorPreview={selectorRulePreview}
        selectorPreviewPending={selectorRulePreviewMutation.isPending}
        selectorMutationPending={
          createSelectorRuleMutation.isPending ||
          deleteSelectorRuleMutation.isPending
        }
        selectorMutationError={
          createSelectorRuleMutation.error ??
          deleteSelectorRuleMutation.error ??
          selectorRulePreviewMutation.error
        }
        policyWritesEnabled={dashboard?.policyWritesEnabled ?? false}
        isLoading={revisionsQuery.isLoading || auditsQuery.isLoading}
        t={t}
        locale={locale}
        onRollback={setRollbackTarget}
        onViewRevision={(revision) => setRevisionDetailId(revision.id)}
        onPreviewSelectorRule={(input) => {
          setSelectorRulePreview(undefined)
          selectorRulePreviewMutation.mutate(input)
        }}
        onCreateSelectorRule={(input) =>
          createSelectorRuleMutation.mutate(input)
        }
        onDeleteSelectorRule={setSelectorRuleDeleteTarget}
      />
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

      {selectorRuleDeleteTarget ? (
        <ConfirmDialog
          title={t("relay.policy.removeSelectorRule")}
          description={`${selectorRuleDeleteTarget.sourceSelector} → ${selectorRuleDeleteTarget.relaySelector}`}
          confirmLabel={t("relay.policy.removeSelectorRule")}
          cancelLabel={t("relay.policy.cancel")}
          isPending={deleteSelectorRuleMutation.isPending}
          error={deleteSelectorRuleMutation.error}
          onCancel={() => setSelectorRuleDeleteTarget(null)}
          onConfirm={() =>
            deleteSelectorRuleMutation.mutate({
              id: selectorRuleDeleteTarget.id,
              basePolicyHash: dashboard?.policyHash ?? "",
            })
          }
        />
      ) : null}

      <Toast message={toast} />
    </>
  )
}
