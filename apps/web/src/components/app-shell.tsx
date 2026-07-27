import { useEffect, type ReactElement, type ReactNode } from 'react'
import {
  Frame,
  FramePanel,
  FrameTitle,
  ScrollArea,
  initTheme,
} from '@escalite/ui'

import { OrgSwitcher } from './org-switcher'
import { SidebarNav } from './sidebar-nav'

type AppShellProps = {
  title: string
  children: ReactNode
}

export function AppShell({ title, children }: AppShellProps): ReactElement {
  useEffect(() => {
    initTheme()
  }, [])

  return (
    <div className="flex h-svh bg-background">
      {/* Desktop sidebar — mobile drawer wired in EL-166 (p-drawer-11). */}
      <aside
        className="hidden w-56 shrink-0 border-e border-sidebar-border bg-sidebar md:flex md:flex-col"
        data-mobile-drawer-target
        data-slot="app-sidebar"
      >
        <SidebarNav />
      </aside>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col p-2">
        <Frame className="flex min-h-0 flex-1 flex-col">
          <FramePanel className="flex shrink-0 items-center justify-between gap-4 px-5 py-3">
            <FrameTitle className="text-base">{title}</FrameTitle>
            <div className="md:hidden">
              <OrgSwitcher />
            </div>
          </FramePanel>
          <FramePanel className="flex min-h-0 flex-1 flex-col p-0">
            <ScrollArea className="min-h-0 flex-1" fill>
              <div className="p-5">{children}</div>
            </ScrollArea>
          </FramePanel>
        </Frame>
      </div>
    </div>
  )
}
