import type { ReactNode } from "react"
import {
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Server,
  Tag,
  type LucideIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

export function Pagination({
  page,
  pageCount,
  onPage,
  t,
}: {
  page: number
  pageCount: number
  onPage: (page: number) => void
  t: (key: string) => string
}) {
  if (pageCount <= 1) return null
  return (
    <nav
      aria-label={t("relay.common.pagination")}
      className="flex items-center justify-center gap-3 border-t border-border px-4 py-3"
    >
      <Button
        variant="outline"
        size="icon-xs"
        isDisabled={page <= 0}
        aria-label={t("relay.common.previousPage")}
        onPress={() => onPage(page - 1)}
      >
        <ChevronLeft size={14} />
      </Button>
      <span className="text-[10px] text-muted-foreground">
        {page + 1} / {pageCount}
      </span>
      <Button
        variant="outline"
        size="icon-xs"
        isDisabled={page >= pageCount - 1}
        aria-label={t("relay.common.nextPage")}
        onPress={() => onPage(page + 1)}
      >
        <ChevronRight size={14} />
      </Button>
    </nav>
  )
}

export function OptionSelect({
  value,
  onValueChange,
  ariaLabel,
  options,
}: {
  value: string
  onValueChange: (value: string) => void
  ariaLabel: string
  options: { value: string; label: string }[]
}) {
  return (
    <Select
      aria-label={ariaLabel}
      selectedKey={value}
      onSelectionChange={(key) => {
        if (typeof key === "string") onValueChange(key)
      }}
    >
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} id={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function Field({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <label className="grid gap-2 text-xs font-medium text-muted-foreground">
      <span>{label}</span>
      {children}
    </label>
  )
}

export function MetricCard({
  icon: Icon,
  label,
  value,
  detail,
  detailValue,
}: {
  icon: LucideIcon
  label: string
  value: number | string
  detail?: string
  detailValue?: number
}) {
  return (
    <Card className="min-h-32 p-4">
      <span className="grid size-9 place-items-center border border-border bg-muted text-muted-foreground">
        <Icon size={17} />
      </span>
      <div className="mt-4 flex items-baseline justify-between gap-2">
        <span className="text-2xl font-semibold tabular-nums">{value}</span>
        <span className="truncate text-xs text-muted-foreground">{label}</span>
      </div>
      {detail ? (
        <p className="mt-1 truncate text-xs text-muted-foreground">
          {detailValue !== undefined ? `${detailValue} · ${detail}` : detail}
        </p>
      ) : null}
    </Card>
  )
}

export function DeviceIcon({
  tags,
  small = false,
}: {
  tags: string[]
  small?: boolean
}) {
  const Icon = tags.length ? Tag : Server
  return (
    <span
      className={cn(
        "grid shrink-0 place-items-center border border-border bg-muted text-muted-foreground",
        small ? "size-8" : "size-10"
      )}
    >
      <Icon size={small ? 14 : 17} />
    </span>
  )
}

export function Detail({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon
  label: string
  value: string
}) {
  return (
    <div className="min-w-0">
      <p className="flex items-center gap-1 text-xs text-muted-foreground">
        <Icon size={12} />
        {label}
      </p>
      <p className="mt-1 truncate text-xs font-medium">{value}</p>
    </div>
  )
}

export function StatusTag({
  status,
  t,
}: {
  status: string
  t: (key: string) => string
}) {
  return (
    <Badge
      variant={
        status === "rejected" || status === "invalidated"
          ? "destructive"
          : "secondary"
      }
    >
      {t(`relay.applications.${status}`)}
    </Badge>
  )
}

export function SourceLegend({
  source,
  t,
}: {
  source: string
  t: (key: string) => string
}) {
  const label =
    source === "platform"
      ? "relay.editor.platform"
      : source === "external"
        ? "relay.editor.external"
        : source === "inherited"
          ? "relay.editor.inherited"
          : "relay.editor.unknown"
  return (
    <span className="inline-flex items-center gap-2">
      <span className="size-2 bg-primary" aria-hidden="true" />
      {t(label)}
    </span>
  )
}

export function SourceTag({
  source,
  t,
}: {
  source: string
  t: (key: string) => string
}) {
  const key =
    source === "platform"
      ? "platform"
      : source === "external"
        ? "external"
        : source === "inherited"
          ? "inherited"
          : source === "exception"
            ? "exception"
            : source === "unknown"
              ? "unknown"
              : "new"
  return (
    <Badge variant={source === "unknown" ? "outline" : "secondary"}>
      {t(`relay.editor.${key}`)}
    </Badge>
  )
}

export function Notice({
  icon: Icon,
  tone,
  text,
}: {
  icon: LucideIcon
  tone: "amber" | "red" | "neutral"
  text: string
}) {
  return (
    <div
      className={cn(
        "mt-4 flex items-start gap-2 border border-border bg-muted px-3 py-2.5 text-xs leading-5 text-muted-foreground",
        tone === "red" && "border-destructive/40 text-destructive"
      )}
    >
      <Icon size={14} className="mt-0.5 shrink-0" />
      <span>{text}</span>
    </div>
  )
}

export function EmptyState({
  title,
  icon: Icon,
}: {
  title: string
  icon: LucideIcon
}) {
  return (
    <div className="flex min-h-32 flex-col items-center justify-center border border-dashed border-border px-5 py-7 text-center">
      <span className="grid size-9 place-items-center border border-border bg-muted text-muted-foreground">
        <Icon size={17} />
      </span>
      <p className="mt-3 max-w-sm text-xs leading-5 text-muted-foreground">
        {title}
      </p>
    </div>
  )
}

export function LoadingState({ t }: { t: (key: string) => string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-xs text-muted-foreground">
      <RefreshCw size={14} className="animate-spin" />
      {t("relay.common.loading")}
    </div>
  )
}

export function HashChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 border border-border bg-muted px-3 py-2">
      <p className="text-[0.65rem] font-semibold tracking-wider text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
        {value}
      </p>
    </div>
  )
}
