import { useEffect, useState, type FormEvent } from "react"
import { Check, ShieldCheck, TriangleAlert } from "lucide-react"

import type {
  PeerRelaySettings,
  PeerRelaySettingsInput,
} from "@/api/generated/model"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Field,
  LoadingState,
  Notice,
  OptionSelect,
} from "@/features/peer-relay/components/view-primitives"
import {
  errorMessage,
  formatDate,
  parseGroupList,
} from "@/features/peer-relay/utils"
import { cn } from "@/lib/utils"

export function SettingsPage({
  settings,
  isLoading,
  error,
  isPending,
  saveError,
  locale,
  t,
  onSave,
}: {
  settings?: PeerRelaySettings
  isLoading: boolean
  error: Error | null
  isPending: boolean
  saveError: Error | null
  locale: string
  t: (key: string) => string
  onSave: (input: PeerRelaySettingsInput) => void
}) {
  const [adminGroups, setAdminGroups] = useState(
    settings?.adminGroups.join(", ") ?? ""
  )
  const [reviewerGroups, setReviewerGroups] = useState(
    settings?.reviewerGroups.join(", ") ?? ""
  )
  const [applicantGroups, setApplicantGroups] = useState(
    settings?.applicantGroups.join(", ") ?? ""
  )
  const [applicantMode, setApplicantMode] = useState<
    PeerRelaySettingsInput["applicantMode"]
  >(settings?.applicantMode ?? "all")
  const [reviewerManageRelays, setReviewerManageRelays] = useState(
    settings?.reviewerManageRelays ?? false
  )
  const [reviewerEditAllDevices, setReviewerEditAllDevices] = useState(
    settings?.reviewerEditAllDevices ?? false
  )
  const [selfServiceEnabled, setSelfServiceEnabled] = useState(
    settings?.selfServiceEnabled ?? true
  )
  const [syncIntervalSeconds, setSyncIntervalSeconds] = useState(
    String(settings?.syncIntervalSeconds ?? 60)
  )
  const [reapplyCooldownHours, setReapplyCooldownHours] = useState(
    String(settings?.reapplyCooldownHours ?? 0)
  )
  useEffect(() => {
    if (!settings) return
    setAdminGroups(settings.adminGroups.join(", "))
    setReviewerGroups(settings.reviewerGroups.join(", "))
    setApplicantGroups(settings.applicantGroups.join(", "))
    setApplicantMode(settings.applicantMode)
    setReviewerManageRelays(settings.reviewerManageRelays)
    setReviewerEditAllDevices(settings.reviewerEditAllDevices)
    setSelfServiceEnabled(settings.selfServiceEnabled)
    setSyncIntervalSeconds(String(settings.syncIntervalSeconds))
    setReapplyCooldownHours(String(settings.reapplyCooldownHours))
  }, [settings])
  const intervalValue = Number(syncIntervalSeconds)
  const cooldownValue = Number(reapplyCooldownHours)
  const validSyncInterval =
    Number.isInteger(intervalValue) &&
    intervalValue >= 10 &&
    intervalValue <= 3600
  const validCooldown =
    Number.isInteger(cooldownValue) &&
    cooldownValue >= 0 &&
    cooldownValue <= 720
  const canSave =
    parseGroupList(adminGroups).length > 0 && validSyncInterval && validCooldown

  if (isLoading) return <LoadingState t={t} />
  if (error || !settings) {
    return (
      <Notice
        icon={TriangleAlert}
        tone="red"
        text={errorMessage(error, t("errors.unknown"))}
      />
    )
  }

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onSave({
      adminGroups: parseGroupList(adminGroups),
      reviewerGroups: parseGroupList(reviewerGroups),
      applicantGroups: parseGroupList(applicantGroups),
      applicantMode,
      reviewerManageRelays,
      reviewerEditAllDevices,
      selfServiceEnabled,
      syncIntervalSeconds: intervalValue,
      reapplyCooldownHours: cooldownValue,
    })
  }

  const groupLabel = (title: string, description: string) => (
    <span>
      <span className="block text-xs font-semibold text-foreground">
        {title}
      </span>
      <span className="mt-1 block text-xs leading-5 text-muted-foreground">
        {description}
      </span>
    </span>
  )

  return (
    <form className="space-y-5" onSubmit={submit}>
      <Card className="p-5 sm:p-6">
        <div className="mb-5 flex items-start gap-3">
          <span className="grid size-9 shrink-0 place-items-center border border-border bg-muted text-muted-foreground">
            <ShieldCheck size={17} />
          </span>
          <div>
            <h3 className="text-sm font-semibold">
              {t("relay.settings.groupsTitle")}
            </h3>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              {t("relay.settings.groupsHelp")}
            </p>
          </div>
        </div>
        <div className="grid gap-4 md:grid-cols-3">
          <Field label={t("relay.settings.adminGroups")}>
            <Textarea
              className="min-h-24 leading-5"
              value={adminGroups}
              onChange={(event) => setAdminGroups(event.target.value)}
              placeholder="group:relay-admin"
              required
            />
          </Field>
          <Field label={t("relay.settings.reviewerGroups")}>
            <Textarea
              className="min-h-24 leading-5"
              value={reviewerGroups}
              onChange={(event) => setReviewerGroups(event.target.value)}
              placeholder="group:relay-reviewer"
            />
          </Field>
          <Field label={t("relay.settings.applicantGroups")}>
            <Textarea
              className="min-h-24 leading-5"
              value={applicantGroups}
              onChange={(event) => setApplicantGroups(event.target.value)}
              placeholder="group:relay-applicant"
            />
          </Field>
        </div>
        <p className="mt-3 text-xs leading-5 text-muted-foreground">
          {t("relay.settings.groupFormat")}
        </p>
      </Card>

      <section className="grid gap-5 xl:grid-cols-2">
        <Card className="p-5 sm:p-6">
          <div className="mb-4">
            <h3 className="text-sm font-semibold">
              {t("relay.settings.applicantTitle")}
            </h3>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              {t("relay.settings.applicantHelp")}
            </p>
          </div>
          <Field label={t("relay.settings.applicantMode")}>
            <OptionSelect
              value={applicantMode}
              onValueChange={(value) =>
                setApplicantMode(
                  value as PeerRelaySettingsInput["applicantMode"]
                )
              }
              ariaLabel={t("relay.settings.applicantMode")}
              options={[
                { value: "all", label: t("relay.settings.modeAll") },
                { value: "groups", label: t("relay.settings.modeGroups") },
                { value: "disabled", label: t("relay.settings.modeDisabled") },
              ]}
            />
          </Field>
          <div className="mt-4 border border-border p-3.5">
            <Checkbox
              isSelected={selfServiceEnabled}
              onChange={setSelfServiceEnabled}
            >
              {groupLabel(
                t("relay.settings.selfServiceTitle"),
                t("relay.settings.selfServiceHelp")
              )}
            </Checkbox>
          </div>
        </Card>

        <Card className="p-5 sm:p-6">
          <div className="mb-4">
            <h3 className="text-sm font-semibold">
              {t("relay.settings.reviewerTitle")}
            </h3>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              {t("relay.settings.reviewerHelp")}
            </p>
          </div>
          <div className="space-y-3">
            <div className="border border-border p-3.5">
              <Checkbox
                isSelected={reviewerManageRelays}
                onChange={setReviewerManageRelays}
              >
                {groupLabel(
                  t("relay.settings.reviewerManageRelays"),
                  t("relay.settings.reviewerManageRelaysHelp")
                )}
              </Checkbox>
            </div>
            <div className="border border-border p-3.5">
              <Checkbox
                isSelected={reviewerEditAllDevices}
                onChange={setReviewerEditAllDevices}
              >
                {groupLabel(
                  t("relay.settings.reviewerEditAllDevices"),
                  t("relay.settings.reviewerEditAllDevicesHelp")
                )}
              </Checkbox>
            </div>
          </div>
        </Card>
      </section>

      <Card className="p-5 sm:p-6">
        <div className="mb-4">
          <h3 className="text-sm font-semibold">
            {t("relay.settings.runtimeTitle")}
          </h3>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            {t("relay.settings.runtimeHelp")}
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <SettingStatus
            label={t("relay.settings.headscale")}
            value={
              settings.headscaleConfigured
                ? t("relay.settings.configured")
                : t("relay.settings.notConfigured")
            }
            active={settings.headscaleConfigured}
          />
          <SettingStatus
            label={t("relay.settings.policyWrites")}
            value={
              settings.policyWritesEnabled
                ? t("relay.settings.enabled")
                : t("relay.settings.disabled")
            }
            active={settings.policyWritesEnabled}
          />
          <Field label={t("relay.settings.syncInterval")}>
            <Input
              type="number"
              min={10}
              max={3600}
              step={1}
              value={syncIntervalSeconds}
              onChange={(event) => setSyncIntervalSeconds(event.target.value)}
              required
              aria-describedby="sync-interval-help"
            />
            <span
              id="sync-interval-help"
              className="mt-1.5 block text-xs text-muted-foreground"
            >
              {t("relay.settings.syncIntervalHelp")}
            </span>
          </Field>
          <Field label={t("relay.settings.reapplyCooldown")}>
            <Input
              type="number"
              min={0}
              max={720}
              step={1}
              value={reapplyCooldownHours}
              onChange={(event) => setReapplyCooldownHours(event.target.value)}
              required
              aria-describedby="reapply-cooldown-help"
            />
            <span
              id="reapply-cooldown-help"
              className="mt-1.5 block text-xs text-muted-foreground"
            >
              {t("relay.settings.reapplyCooldownHelp")}
            </span>
          </Field>
          <SettingStatus
            label={t("relay.settings.lastUpdated")}
            value={
              settings.updatedAt
                ? formatDate(settings.updatedAt, locale)
                : t("relay.settings.environmentDefault")
            }
            active={false}
          />
        </div>
      </Card>

      {saveError ? (
        <Notice
          icon={TriangleAlert}
          tone="red"
          text={errorMessage(saveError, t("errors.unknown"))}
        />
      ) : null}
      <div className="flex flex-col-reverse gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {settings.updatedBy
            ? `${t("relay.settings.updatedBy")}: ${settings.updatedBy}`
            : t("relay.settings.environmentDefault")}
        </p>
        <Button type="submit" isDisabled={isPending || !canSave}>
          <Check size={14} />
          {isPending ? t("relay.editor.saving") : t("relay.settings.save")}
        </Button>
      </div>
    </form>
  )
}

function SettingStatus({
  label,
  value,
  active,
}: {
  label: string
  value: string
  active: boolean
}) {
  return (
    <div className="border border-border bg-muted p-3.5">
      <p className="text-xs font-semibold tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className="mt-2 flex items-center gap-2 text-sm font-semibold">
        <span
          className={cn(
            "size-1.5",
            active ? "bg-primary" : "bg-muted-foreground"
          )}
        />
        {value}
      </p>
    </div>
  )
}
