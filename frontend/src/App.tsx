import { LanguageToggle } from "@/components/language-toggle"
import { ModeToggle } from "@/components/mode-toggle"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n"

export function App() {
  const { t } = useI18n()

  return (
    <div className="flex min-h-svh p-6">
      <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
        <div className="flex justify-end gap-2">
          <LanguageToggle />
          <ModeToggle />
        </div>
        <div>
          <h1 className="font-medium">{t("home.title")}</h1>
          <p>{t("home.body")}</p>
          <Button className="mt-2">{t("app.home")}</Button>
        </div>
      </div>
    </div>
  )
}

export default App
