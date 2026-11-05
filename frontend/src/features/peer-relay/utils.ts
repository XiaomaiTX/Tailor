import type { Relay, RelayUpdate } from "@/api/generated/model"
import { ApiProblemError } from "@/api/http"

export function optional(value: string) {
  const normalized = value.trim()
  return normalized || undefined
}

export function parseGroupList(value: string) {
  return [
    ...new Set(
      value
        .split(/[\n,]/)
        .map((group) => group.trim())
        .filter(Boolean)
    ),
  ]
}

export function relayUpdateInput(
  relay: Relay,
  selectable: boolean
): RelayUpdate {
  return {
    displayName: relay.displayName,
    region: relay.region,
    udpPort: relay.udpPort,
    publicEndpoint: relay.publicEndpoint,
    isp: relay.isp,
    description: relay.description,
    selectable,
  }
}

export function errorMessage(error: unknown, fallback: string) {
  if (error instanceof ApiProblemError) return error.message || fallback
  if (error instanceof Error) return error.message || fallback
  return fallback
}

export function formatDate(value: string, locale: string) {
  const date = new Date(value)
  if (Number.isNaN(date.valueOf())) return value
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}
