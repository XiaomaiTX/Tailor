import { ArrowRight, RadioTower, ShieldCheck } from "lucide-react"
import { getApiBaseUrl } from "@/api/http"
import { LanguageToggle } from "@/components/language-toggle"
import { ModeToggle } from "@/components/mode-toggle"
import { useI18n } from "@/i18n"

export function OidcLoginPage() {
  const { t } = useI18n()
  const apiBase = getApiBaseUrl().replace(/\/$/, "")
  const loginURL = `${apiBase}/auth/login`
  const failed = new URLSearchParams(window.location.search).get("error") === "sso"

  return (
    <main className="grid min-h-svh place-items-center bg-background px-5 py-12 text-foreground">
      <section className="w-full max-w-md border border-border bg-card p-8 shadow-sm sm:p-10">
        <div className="flex items-center justify-between">
          <div className="grid size-11 place-items-center bg-primary text-primary-foreground">
            <RadioTower size={21} />
          </div>
          <div className="flex items-center gap-2">
            <LanguageToggle />
            <ModeToggle />
          </div>
        </div>
        <p className="mt-9 text-[10px] font-bold tracking-[0.2em] text-primary">
          {t("relay.brand")}
        </p>
        <p className="mt-1 text-xs leading-5 text-muted-foreground">
          {t("relay.tagline")}
        </p>
        <h1 className="mt-2 font-heading text-2xl font-semibold tracking-tight">
          {t("auth.title")}
        </h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          {t("auth.description")}
        </p>
        {failed ? (
          <p role="alert" className="mt-5 border border-destructive/30 bg-destructive/5 px-3 py-2.5 text-sm text-destructive">
            {t("auth.failed")}
          </p>
        ) : null}
        <a
          href={loginURL}
          className="mt-7 inline-flex h-11 w-full items-center justify-center gap-2 bg-primary px-4 text-xs font-bold tracking-widest text-primary-foreground uppercase transition hover:bg-primary/85 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          <ShieldCheck size={16} />
          {t("auth.login")}
          <ArrowRight size={15} />
        </a>
      </section>
    </main>
  )
}
