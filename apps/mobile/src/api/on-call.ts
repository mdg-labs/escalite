import { graphqlRequest } from '@/api/graphql'

export type MyOnCallAssignment = {
  scheduleId: string
  scheduleName: string
  teamName: string
  layer: number
  until: string
}

const assignmentFields = `
  scheduleId
  scheduleName
  teamName
  layer
  until
`

export function parseMyOnCallAssignments(
  assignments: MyOnCallAssignment[] | null | undefined,
): MyOnCallAssignment[] {
  if (!assignments?.length) {
    return []
  }

  return assignments
}

export function isOnCall(assignments: MyOnCallAssignment[]): boolean {
  return assignments.length > 0
}

export async function fetchMyOnCallStatus(): Promise<MyOnCallAssignment[]> {
  const data = await graphqlRequest<{ myOnCallStatus: MyOnCallAssignment[] }>(
    `query MyOnCallStatus {
      myOnCallStatus {
        ${assignmentFields}
      }
    }`,
  )

  return parseMyOnCallAssignments(data.myOnCallStatus)
}
