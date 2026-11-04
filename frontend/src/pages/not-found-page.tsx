import { Link } from "react-router-dom"
import { useI18n } from "@/i18n"

export function NotFoundPage() {
  const { t } = useI18n()

  return (
    <main className="flex min-h-svh items-center justify-center p-6">
      <section className="max-w-md space-y-4 text-center">
        <h1 className="text-3xl font-medium">{t("notFound.title")}</h1>
        <p className="text-muted-foreground">{t("notFound.body")}</p>
        <Link className="text-sm underline underline-offset-4" to="/">
          {t("notFound.back")}
        </Link>
      </section>
    </main>
  )
}
