import { useI18n } from "@/i18n"
import { Check } from "lucide-react"
import { type ReactNode } from "react"
export function PageHeading({
  section,
  children,
}: {
  section: string
  children?: ReactNode
}) {
  const { t } = useI18n()
  return (
    <div className="mb-7 flex flex-col justify-between gap-4 sm:mb-8 sm:flex-row sm:items-end">
      <div>
        <h2 className="font-heading text-[28px] font-medium tracking-[-0.035em] sm:text-[34px]">
          {t(`relay.${section}.title`)}
        </h2>
        <p className="mt-2 max-w-[760px] text-sm leading-6 text-muted-foreground">
          {t(`relay.${section}.description`)}
        </p>
      </div>
      {children}
    </div>
  )
}
export function Toast({ message }: { message: string }) {
  return message ? (
    <div
      role="status"
      className="fixed bottom-5 left-1/2 z-50 flex -translate-x-1/2 items-center gap-2 border border-border bg-primary px-4 py-2.5 text-xs font-semibold text-primary-foreground shadow-xl"
    >
      <Check size={14} />
      {message}
    </div>
  ) : null
}
