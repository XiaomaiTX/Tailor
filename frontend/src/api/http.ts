declare const __API_BASE_URL__: string

export interface ProblemDetails {
  type: string
  title: string
  status: number
  detail?: string
  instance?: string
}

export class ApiProblemError extends Error {
  readonly problem: ProblemDetails

  constructor(problem: ProblemDetails) {
    super(problem.detail || problem.title)
    this.name = "ApiProblemError"
    this.problem = problem
  }
}

export function getApiBaseUrl(): string {
  return typeof __API_BASE_URL__ === "string" ? __API_BASE_URL__ : "/api/v1"
}

function isProblemDetails(value: unknown): value is ProblemDetails {
  if (typeof value !== "object" || value === null) return false

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.type === "string" &&
    typeof candidate.title === "string" &&
    typeof candidate.status === "number"
  )
}

export async function customFetch<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const baseURL = getApiBaseUrl().replace(/\/$/, "")
  const target = /^https?:\/\//.test(url)
    ? url
    : `${baseURL}/${url.replace(/^\//, "")}`
  const headers = new Headers(options.headers)
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json")
  }
  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(target, {
    ...options,
    headers,
  })
  const body: unknown =
    response.status === 204 ? undefined : await response.json()
  if (!response.ok) {
    if (isProblemDetails(body)) {
      throw new ApiProblemError(body)
    }
    throw new ApiProblemError({
      type: "urn:go-react-starter:problem:unknown",
      title: "Request failed",
      status: response.status,
    })
  }
  return { data: body, status: response.status, headers: response.headers } as T
}
