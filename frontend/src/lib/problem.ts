import { ApiProblemError } from "@/api/http"

export function problemMessageKey(error: unknown) {
  if (!(error instanceof ApiProblemError)) return "errors.unknown"

  switch (error.problem.status) {
    case 400:
      return "errors.validation"
    case 404:
      return "errors.notFound"
    case 500:
      return "errors.internal"
    default:
      return "errors.unknown"
  }
}
