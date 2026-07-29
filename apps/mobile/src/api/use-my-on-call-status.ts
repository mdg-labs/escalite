import { useFocusEffect } from 'expo-router'
import { useCallback, useState } from 'react'

import { fetchMyOnCallStatus, type MyOnCallAssignment } from '@/api/on-call'

type UseMyOnCallStatusResult = {
  assignments: MyOnCallAssignment[]
  loading: boolean
  refreshing: boolean
  error: string | null
  refresh: () => Promise<void>
}

export function useMyOnCallStatus(enabled: boolean): UseMyOnCallStatusResult {
  const [assignments, setAssignments] = useState<MyOnCallAssignment[]>([])
  const [loading, setLoading] = useState(enabled)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(
    async (isRefresh = false) => {
      if (!enabled) {
        setLoading(false)
        return
      }

      if (isRefresh) {
        setRefreshing(true)
      } else {
        setLoading(true)
      }
      setError(null)

      try {
        const nextAssignments = await fetchMyOnCallStatus()
        setAssignments(nextAssignments)
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Unable to load on-call status')
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [enabled],
  )

  const refresh = useCallback(async () => {
    await load(true)
  }, [load])

  useFocusEffect(
    useCallback(() => {
      void load()
    }, [load]),
  )

  return {
    assignments,
    loading,
    refreshing,
    error,
    refresh,
  }
}
