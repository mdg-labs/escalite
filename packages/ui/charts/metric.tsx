import type { ReactElement } from 'react'
import { Metric as TremorMetric, type MetricProps } from '@tremor/react'

export type { MetricProps }

export function Metric(props: MetricProps): ReactElement {
  return <TremorMetric {...props} />
}
