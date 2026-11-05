import { useState } from "react"
import { Bell, MapPin } from "lucide-react"
import type { Device, RelayApplication } from "@/api/generated/model"
import { Button } from "@/components/ui/button"
import {
  DeviceIcon,
  EmptyState,
  LoadingState,
  Pagination,
  StatusTag,
} from "@/features/peer-relay/components/view-primitives"

export function ApplicationsPage({
  applications,
  devices,
  isReviewer,
  canApply,
  isLoading,
  t,
  onCreate,
  onCancel,
  onReview,
}: {
  applications: RelayApplication[]
  devices: Device[]
  isReviewer: boolean
  canApply: boolean
  isLoading: boolean
  t: (key: string) => string
  onCreate: () => void
  onCancel: (application: RelayApplication) => void
  onReview: (
    application: RelayApplication,
    decision: "approved" | "rejected"
  ) => void
}) {
  const [page, setPage] = useState(0)
  const pageSize = 20
  const pageCount = Math.max(1, Math.ceil(applications.length / pageSize))
  const currentPage = Math.min(page, pageCount - 1)
  const visibleApplications = applications.slice(
    currentPage * pageSize,
    currentPage * pageSize + pageSize
  )
  return (
    <div className="space-y-4">
      {canApply ? (
        <div className="flex flex-col justify-between gap-3 border border-border bg-card p-4 sm:flex-row sm:items-center">
          {devices.length === 0 ? (
            <p className="text-xs text-muted-foreground">
              {t("relay.applications.noEligibleDevice")}
            </p>
          ) : null}
          <Button
            isDisabled={!devices.length}
            onPress={onCreate}
            className="h-9 shrink-0 px-4"
          >
            {t("relay.applications.create")}
          </Button>
        </div>
      ) : null}
      {isLoading ? (
        <LoadingState t={t} />
      ) : applications.length === 0 ? (
        <EmptyState title={t("relay.applications.empty")} icon={Bell} />
      ) : (
        <div className="space-y-3">
          {visibleApplications.map((application) => (
            <article
              key={application.id}
              className="rounded-none border border-border bg-card p-4 sm:p-5"
            >
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
                <DeviceIcon tags={application.device.tags} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-sm font-semibold">
                      {application.displayName}
                    </h3>
                    <StatusTag status={application.status} t={t} />
                  </div>
                  <p className="mt-1.5 text-[11px] text-muted-foreground">
                    {application.device.givenName ||
                      application.device.hostname}{" "}
                    <span className="mx-1">·</span>
                    {application.device.ipv4 ||
                      application.device.ipv6 ||
                      "—"}{" "}
                    <span className="mx-1">·</span>
                    {application.applicant}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-[10px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1">
                      <MapPin size={12} />
                      {application.region || t("relay.common.region")}
                    </span>
                    {application.udpPort ? (
                      <span>UDP {application.udpPort}</span>
                    ) : null}
                    {application.publicEndpoint ? (
                      <span>{application.publicEndpoint}</span>
                    ) : null}
                    {application.isp ? <span>{application.isp}</span> : null}
                  </div>
                  {application.reason ? (
                    <p className="mt-3 max-w-3xl text-xs leading-5 text-muted-foreground">
                      {application.reason}
                    </p>
                  ) : null}
                  {application.reviewNote ? (
                    <p className="mt-3 rounded-none bg-muted px-3 py-2 text-[11px] leading-5 text-muted-foreground">
                      {application.reviewNote}
                    </p>
                  ) : null}
                </div>
                {isReviewer && application.status === "pending" ? (
                  <div className="flex shrink-0 gap-2 sm:flex-col">
                    <Button
                      onPress={() => onReview(application, "approved")}
                      className="h-9 px-3"
                    >
                      {t("relay.applications.approve")}
                    </Button>
                    <Button
                      variant="destructive"
                      onPress={() => onReview(application, "rejected")}
                      className="h-9 px-3"
                    >
                      {t("relay.applications.reject")}
                    </Button>
                  </div>
                ) : null}
                {!isReviewer && application.status === "pending" ? (
                  <Button
                    variant="outline"
                    onPress={() => onCancel(application)}
                    className="h-9 shrink-0 px-3"
                  >
                    {t("relay.applications.withdraw")}
                  </Button>
                ) : null}
              </div>
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
