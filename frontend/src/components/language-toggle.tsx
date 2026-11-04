import { Languages } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useI18n } from "@/i18n"

export function LanguageToggle() {
  const { setLocale, t } = useI18n()

  return (
    <DropdownMenuTrigger>
      <Button
        aria-label={t("preferences.languageMenu")}
        size="icon"
        variant="outline"
      >
        <Languages className="size-4" />
      </Button>
      <DropdownMenu placement="bottom end">
        <DropdownMenuItem onAction={() => setLocale("zh-CN")}>
          {t("preferences.chinese")}
        </DropdownMenuItem>
        <DropdownMenuItem onAction={() => setLocale("en-US")}>
          {t("preferences.english")}
        </DropdownMenuItem>
      </DropdownMenu>
    </DropdownMenuTrigger>
  )
}
