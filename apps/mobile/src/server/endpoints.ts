import type { ServerEndpoints } from '@/server/types'
import { defaultDevEndpoints } from '@/server/url'

let activeEndpoints: ServerEndpoints = defaultDevEndpoints()

export function getServerEndpoints(): ServerEndpoints {
  return activeEndpoints
}

export function setActiveServerEndpoints(endpoints: ServerEndpoints): void {
  activeEndpoints = endpoints
}

export function resetActiveServerEndpoints(): void {
  activeEndpoints = defaultDevEndpoints()
}
