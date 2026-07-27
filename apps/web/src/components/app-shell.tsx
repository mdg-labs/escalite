import { useEffect, type ReactElement, type ReactNode } from 'react'
import {
  Frame,
  FramePanel,
  FrameTitle,
  ScrollArea,
  initTheme,
} from '@escalite/ui'

import { OrgSwitcher } from './org-switcher'
import { MobileSidebarDrawer, SidebarNav } from './sidebar-nav'

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
      <aside
        className="hidden w-56 shrink-0 border-e border-sidebar-border bg-sidebar md:flex md:flex-col"
        data-slot="app-sidebar"
      >
        <SidebarNav />
      </aside>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col p-2">
        <Frame className="flex min-h-0 flex-1 flex-col">
          <FramePanel className="flex shrink-0 items-center justify-between gap-4 px-5 py-3">
            <div className="flex min-w-0 flex-1 items-center gap-3">
              <div className="md:hidden">
                <MobileSidebarDrawer />
              </div>
              <FrameTitle className="min-w-0 truncate text-base">{title}</FrameTitle>
            </div>
            <div className="shrink-0 md:hidden">
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
