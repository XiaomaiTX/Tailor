import { useState } from "react"
import { CircleHelp, Server } from "lucide-react"

import type {
  Device,
  RelayApplication,
  RelayRegistration,
  ReviewInput,
} from "@/api/generated/model"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  EmptyState,
  Field,
  Notice,
  OptionSelect,
} from "@/features/peer-relay/components/view-primitives"
import { errorMessage, optional } from "@/features/peer-relay/utils"

function DialogHeader({ label }: { label: string }) {
  return (
    <div className="mb-5">
      <DialogTitle className="text-base">{label}</DialogTitle>
    </div>
  )
}

function DialogActions({
  cancelLabel,
  confirmLabel,
  isPending,
  onClose,
  disabled = false,
  variant = "default",
}: {
  cancelLabel: string
  confirmLabel: string
  isPending: boolean
  onClose: () => void
  disabled?: boolean
  variant?: "default" | "destructive"
}) {
  return (
    <div className="flex justify-end gap-2 pt-2">
      <Button
        type="button"
        variant="ghost"
        onPress={onClose}
        isDisabled={isPending}
      >
        {cancelLabel}
      </Button>
      <Button
        type="submit"
        variant={variant}
        isDisabled={isPending || disabled}
      >
        {confirmLabel}
      </Button>
    </div>
  )
}

export function ApplicationDialog({
  devices,
  t,
  isPending,
  error,
  onClose,
  onSubmit,
}: {
  devices: Device[]
  t: (key: string) => string
  isPending: boolean
  error: Error | null
  onClose: () => void
  onSubmit: (input: {
    nodeId: string
    displayName: string
    region?: string
    udpPort?: number
    publicEndpoint?: string
    isp?: string
    reason?: string
  }) => void
}) {
  const [nodeId, setNodeId] = useState(devices[0]?.nodeId ?? "")
  const [displayName, setDisplayName] = useState("")
  const [region, setRegion] = useState("")
  const [port, setPort] = useState("")
  const [endpoint, setEndpoint] = useState("")
  const [isp, setIsp] = useState("")
  const [reason, setReason] = useState("")
  const title = t("relay.applications.create")

  return (
    <Dialog
      isOpen
      closeLabel={t("relay.common.close")}
      onOpenChange={(open) => !open && onClose()}
    >
      <DialogHeader label={title} />
      {devices.length === 0 ? (
        <Notice
          icon={CircleHelp}
          tone="neutral"
          text={t("relay.applications.noEligibleDevice")}
        />
      ) : (
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit({
              nodeId,
              displayName,
              region: optional(region),
              udpPort: port ? Number(port) : undefined,
              publicEndpoint: optional(endpoint),
              isp: optional(isp),
              reason: optional(reason),
            })
          }}
        >
          <Field label={t("relay.applications.device")}>
            <OptionSelect
              value={nodeId}
              onValueChange={setNodeId}
              ariaLabel={t("relay.applications.device")}
              options={devices.map((device) => ({
                value: device.nodeId,
                label: `${device.givenName || device.hostname} · ${device.ipv4 || device.ipv6}`,
              }))}
            />
          </Field>
          <Field label={t("relay.relays.displayName")}>
            <Input
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              required
              maxLength={80}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label={t("relay.relays.region")}>
              <Input
                value={region}
                onChange={(event) => setRegion(event.target.value)}
                maxLength={120}
              />
            </Field>
            <Field label={t("relay.relays.port")}>
              <Input
                value={port}
                onChange={(event) => setPort(event.target.value)}
                inputMode="numeric"
                type="number"
                min={1}
                max={65535}
              />
            </Field>
          </div>
          <Field label={t("relay.relays.endpoint")}>
            <Input
              value={endpoint}
              onChange={(event) => setEndpoint(event.target.value)}
              maxLength={253}
              placeholder="relay.example.net"
            />
          </Field>
          <Field label={t("relay.relays.isp")}>
            <Input
              value={isp}
              onChange={(event) => setIsp(event.target.value)}
              maxLength={120}
            />
          </Field>
          <Field label={t("relay.applications.reason")}>
            <Textarea
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              maxLength={2000}
            />
          </Field>
          {error ? (
            <p className="text-sm text-destructive">
              {errorMessage(error, t("errors.unknown"))}
            </p>
          ) : null}
          <DialogActions
            cancelLabel={t("relay.common.cancel")}
            confirmLabel={
              isPending
                ? t("relay.editor.saving")
                : t("relay.applications.submit")
            }
            isPending={isPending}
            onClose={onClose}
            disabled={!nodeId || !displayName.trim()}
          />
        </form>
      )}
    </Dialog>
  )
}

export function RelayRegistrationDialog({
  devices,
  t,
  isPending,
  error,
  onClose,
  onSubmit,
}: {
  devices: Device[]
  t: (key: string) => string
  isPending: boolean
  error: Error | null
  onClose: () => void
  onSubmit: (input: RelayRegistration) => void
}) {
  const [nodeId, setNodeId] = useState(devices[0]?.nodeId ?? "")
  const [displayName, setDisplayName] = useState("")
  const [region, setRegion] = useState("")
  const [port, setPort] = useState("")
  const [endpoint, setEndpoint] = useState("")
  const [isp, setIsp] = useState("")
  const [description, setDescription] = useState("")
  const [selectable, setSelectable] = useState(true)
  const title = t("relay.relays.add")

  return (
    <Dialog
      isOpen
      closeLabel={t("relay.common.close")}
      onOpenChange={(open) => !open && onClose()}
    >
      <DialogHeader label={title} />
      {devices.length === 0 ? (
        <EmptyState
          title={t("relay.relays.noEligibleDevice")}
          icon={Server}
        />
      ) : (
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit({
              nodeId,
              displayName,
              region: optional(region),
              udpPort: port ? Number(port) : undefined,
              publicEndpoint: optional(endpoint),
              isp: optional(isp),
              description: optional(description),
              selectable,
            })
          }}
        >
          <Field label={t("relay.relays.node")}>
            <OptionSelect
              value={nodeId}
              onValueChange={setNodeId}
              ariaLabel={t("relay.relays.node")}
              options={devices.map((device) => ({
                value: device.nodeId,
                label: `${device.givenName || device.hostname} · ${device.username}`,
              }))}
            />
          </Field>
          <Field label={t("relay.relays.displayName")}>
            <Input
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              required
              maxLength={80}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label={t("relay.relays.region")}>
              <Input
                value={region}
                onChange={(event) => setRegion(event.target.value)}
                maxLength={120}
              />
            </Field>
            <Field label={t("relay.relays.port")}>
              <Input
                value={port}
                onChange={(event) => setPort(event.target.value)}
                type="number"
                min={1}
                max={65535}
              />
            </Field>
          </div>
          <Field label={t("relay.relays.endpoint")}>
            <Input
              value={endpoint}
              onChange={(event) => setEndpoint(event.target.value)}
              maxLength={253}
            />
          </Field>
          <Field label={t("relay.relays.isp")}>
            <Input
              value={isp}
              onChange={(event) => setIsp(event.target.value)}
              maxLength={120}
            />
          </Field>
          <Field label={t("relay.relays.descriptionLabel")}>
            <Textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              maxLength={2000}
            />
          </Field>
          <Checkbox isSelected={selectable} onChange={setSelectable}>
            {t("relay.relays.selectable")}
          </Checkbox>
          {error ? (
            <p className="text-sm text-destructive">
              {errorMessage(error, t("errors.unknown"))}
            </p>
          ) : null}
          <DialogActions
            cancelLabel={t("relay.common.cancel")}
            confirmLabel={
              isPending ? t("relay.editor.saving") : t("relay.relays.save")
            }
            isPending={isPending}
            onClose={onClose}
            disabled={!nodeId || !displayName.trim()}
          />
        </form>
      )}
    </Dialog>
  )
}

export function ReviewDialog({
  target,
  note,
  setNote,
  t,
  isPending,
  error,
  onClose,
  onSubmit,
}: {
  target: { application: RelayApplication; decision: "approved" | "rejected" }
  note: string
  setNote: (value: string) => void
  t: (key: string) => string
  isPending: boolean
  error: Error | null
  onClose: () => void
  onSubmit: (input: ReviewInput) => void
}) {
  const rejected = target.decision === "rejected"
  const [overrideEnabled, setOverrideEnabled] = useState(false)
  const [displayName, setDisplayName] = useState(target.application.displayName)
  const [region, setRegion] = useState(target.application.region ?? "")
  const [port, setPort] = useState(
    target.application.udpPort ? String(target.application.udpPort) : ""
  )
  const [endpoint, setEndpoint] = useState(
    target.application.publicEndpoint ?? ""
  )
  const [isp, setIsp] = useState(target.application.isp ?? "")
  const [description, setDescription] = useState(
    target.application.reason ?? ""
  )
  const title = rejected
    ? t("relay.applications.reject")
    : t("relay.applications.approve")

  return (
    <Dialog
      isOpen
      closeLabel={t("relay.common.close")}
      onOpenChange={(open) => !open && onClose()}
    >
      <DialogHeader label={title} />
      <div className="border border-border bg-muted p-3">
        <p className="text-sm font-semibold">
          {target.application.displayName}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {target.application.device.givenName ||
            target.application.device.hostname}{" "}
          · {target.application.applicant}
        </p>
      </div>
      {rejected ? (
        <Field label={t("relay.applications.rejectionReason")}>
          <Textarea
            value={note}
            onChange={(event) => setNote(event.target.value)}
            maxLength={2000}
            required
          />
        </Field>
      ) : (
        <>
          <Checkbox
            className="mt-4"
            isSelected={overrideEnabled}
            onChange={setOverrideEnabled}
          >
            {t("relay.applications.overrideMetadata")}
          </Checkbox>
          {overrideEnabled ? (
            <div className="mt-3 space-y-4 border border-border p-4">
              <p className="text-xs leading-5 text-muted-foreground">
                {t("relay.applications.overrideHelp")}
              </p>
              <Field label={t("relay.relays.displayName")}>
                <Input
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  maxLength={80}
                  required
                />
              </Field>
              <div className="grid grid-cols-2 gap-3">
                <Field label={t("relay.relays.region")}>
                  <Input
                    value={region}
                    onChange={(event) => setRegion(event.target.value)}
                    maxLength={120}
                  />
                </Field>
                <Field label={t("relay.relays.port")}>
                  <Input
                    value={port}
                    onChange={(event) => setPort(event.target.value)}
                    type="number"
                    min={1}
                    max={65535}
                  />
                </Field>
              </div>
              <Field label={t("relay.relays.endpoint")}>
                <Input
                  value={endpoint}
                  onChange={(event) => setEndpoint(event.target.value)}
                  maxLength={253}
                />
              </Field>
              <Field label={t("relay.relays.isp")}>
                <Input
                  value={isp}
                  onChange={(event) => setIsp(event.target.value)}
                  maxLength={120}
                />
              </Field>
              <Field label={t("relay.relays.descriptionLabel")}>
                <Textarea
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  maxLength={2000}
                />
              </Field>
            </div>
          ) : null}
        </>
      )}
      {error ? (
        <p className="mt-3 text-sm text-destructive">
          {errorMessage(error, t("errors.unknown"))}
        </p>
      ) : null}
      <div className="mt-5 flex justify-end gap-2">
        <Button
          type="button"
          variant="ghost"
          onPress={onClose}
          isDisabled={isPending}
        >
          {t("relay.common.cancel")}
        </Button>
        <Button
          type="button"
          variant={rejected ? "destructive" : "default"}
          isDisabled={
            isPending ||
            (rejected && !note.trim()) ||
            (!rejected && overrideEnabled && !displayName.trim())
          }
          onPress={() =>
            onSubmit({
              decision: target.decision,
              note: optional(note),
              relay:
                !rejected && overrideEnabled
                  ? {
                      displayName,
                      region: optional(region),
                      udpPort: port ? Number(port) : null,
                      publicEndpoint: optional(endpoint),
                      isp: optional(isp),
                      description: optional(description),
                    }
                  : undefined,
            })
          }
        >
          {isPending ? t("relay.editor.saving") : title}
        </Button>
      </div>
    </Dialog>
  )
}

export function ConfirmDialog({
  title,
  description,
  confirmLabel,
  cancelLabel,
  isPending,
  error,
  onCancel,
  onConfirm,
}: {
  title: string
  description: string
  confirmLabel: string
  cancelLabel: string
  isPending: boolean
  error: Error | null
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <Dialog
      isOpen
      closeLabel={t("relay.common.close")}
      onOpenChange={(open) => !open && onCancel()}
    >
      <DialogHeader label={title} />
      <p className="text-sm leading-5 text-muted-foreground">{description}</p>
      {error ? (
        <p className="mt-3 text-sm text-destructive">
          {errorMessage(error, "")}
        </p>
      ) : null}
      <div className="mt-5 flex justify-end gap-2">
        <Button
          type="button"
          variant="ghost"
          onPress={onCancel}
          isDisabled={isPending}
        >
          {cancelLabel}
        </Button>
        <Button type="button" onPress={onConfirm} isDisabled={isPending}>
          {isPending ? "…" : confirmLabel}
        </Button>
      </div>
    </Dialog>
  )
}
