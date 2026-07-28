import type { ReactElement } from 'react'
import { Navigate, useSearchParams } from 'react-router'

/** Legacy /integrations — integrations live on the service detail Integrations tab. */
export function IntegrationsPage(): ReactElement {
  const [searchParams] = useSearchParams()
  const serviceId = searchParams.get('serviceId')

  if (serviceId) {
    return <Navigate replace to={`/services/${serviceId}?tab=integrations`} />
  }

  return <Navigate replace to="/services" />
}
