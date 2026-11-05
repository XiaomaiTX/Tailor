import { useEffect, useState } from "react"
export function useToast() {
  const [toast, setToast] = useState("")
  useEffect(() => {
    if (!toast) return
    const timeout = window.setTimeout(() => setToast(""), 3200)
    return () => window.clearTimeout(timeout)
  }, [toast])
  return { toast, setToast }
}
