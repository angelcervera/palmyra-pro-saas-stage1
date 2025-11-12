import type { PropsWithChildren } from 'react'
import { ThemeProvider as NextThemes } from 'next-themes'

export function ThemeProvider({ children }: PropsWithChildren) {
  return (
    <NextThemes attribute="class" defaultTheme="system" enableSystem storageKey="web-admin-theme">
      {children}
    </NextThemes>
  )
}
