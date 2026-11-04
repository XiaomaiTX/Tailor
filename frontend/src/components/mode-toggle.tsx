import { Moon, Sun } from "lucide-react"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useI18n } from "@/i18n"

export function ModeToggle() {
  const { setTheme } = useTheme()
  const { t } = useI18n()

  return (
    <DropdownMenuTrigger>
      <Button
        aria-label={t("preferences.themeMenu")}
        size="icon"
        variant="outline"
      >
        <Sun className="size-4 scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
        <Moon className="absolute size-4 scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
      </Button>
      <DropdownMenu placement="bottom end">
        <DropdownMenuItem onAction={() => setTheme("light")}>
          {t("preferences.light")}
        </DropdownMenuItem>
        <DropdownMenuItem onAction={() => setTheme("dark")}>
          {t("preferences.dark")}
        </DropdownMenuItem>
        <DropdownMenuItem onAction={() => setTheme("system")}>
          {t("preferences.system")}
        </DropdownMenuItem>
      </DropdownMenu>
    </DropdownMenuTrigger>
  )
}
