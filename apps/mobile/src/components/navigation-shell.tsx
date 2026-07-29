import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { Pressable } from 'react-native'
import { Text, YStack } from 'tamagui'

import { useAuth } from '@/auth/context'
import { NavigationDrawer } from '@/components/navigation-drawer'
import { useServerConfig } from '@/server/context'

type NavigationShellContextValue = {
  openDrawer: () => void
  closeDrawer: () => void
}

const NavigationShellContext = createContext<NavigationShellContextValue | null>(null)

type NavigationShellProviderProps = {
  children: ReactNode
}

function HamburgerIcon() {
  return (
    <YStack gap={4} width={22} paddingVertical={4}>
      <YStack height={2} backgroundColor="$color" borderRadius={1} />
      <YStack height={2} backgroundColor="$color" borderRadius={1} />
      <YStack height={2} backgroundColor="$color" borderRadius={1} />
    </YStack>
  )
}

export function NavigationShellProvider({ children }: NavigationShellProviderProps) {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const { user, signOut } = useAuth()
  const { endpoints, clearServer } = useServerConfig()

  const openDrawer = useCallback(() => {
    setDrawerOpen(true)
  }, [])

  const closeDrawer = useCallback(() => {
    setDrawerOpen(false)
  }, [])

  const handleSignOut = useCallback(async () => {
    closeDrawer()
    await signOut()
  }, [closeDrawer, signOut])

  const handleChangeServer = useCallback(async () => {
    closeDrawer()
    await signOut()
    await clearServer()
  }, [clearServer, closeDrawer, signOut])

  const value = useMemo<NavigationShellContextValue>(
    () => ({
      openDrawer,
      closeDrawer,
    }),
    [closeDrawer, openDrawer],
  )

  return (
    <NavigationShellContext.Provider value={value}>
      {children}
      <NavigationDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        email={user?.email ?? null}
        serverOrigin={endpoints?.origin ?? null}
        onSignOut={handleSignOut}
        onChangeServer={handleChangeServer}
      />
    </NavigationShellContext.Provider>
  )
}

export function useNavigationShell(): NavigationShellContextValue {
  const context = useContext(NavigationShellContext)
  if (!context) {
    throw new Error('useNavigationShell must be used within NavigationShellProvider')
  }
  return context
}

export function HeaderBrandTitle() {
  return (
    <Text color="$brand" fontSize="$6" fontWeight="700">
      Escalite
    </Text>
  )
}

export function HeaderMenuButton() {
  const { status } = useAuth()
  const { openDrawer } = useNavigationShell()

  if (status !== 'authenticated') {
    return null
  }

  return (
    <Pressable
      onPress={openDrawer}
      hitSlop={8}
      accessibilityLabel="Open menu"
      accessibilityRole="button"
      style={{ marginLeft: 16 }}
    >
      <HamburgerIcon />
    </Pressable>
  )
}
