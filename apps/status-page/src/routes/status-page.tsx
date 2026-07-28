import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Badge,
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
  Frame,
  FrameDescription,
  FrameHeader,
  FramePanel,
  FrameTitle,
} from '@escalite/ui'
import { AlertCircleIcon, ChevronDownIcon } from 'lucide-react'

import { ComponentStatusBadge } from '../components/component-status-badge'
import { SubscribeForm } from '../components/subscribe-form'
import { statusPageConfig } from '../lib/config'
import { formatDateTime } from '../lib/format'
import { t } from '../lib/i18n'
import {
  fetchPublicStatusPage,
  incidentStatusBadgeVariant,
  incidentStatusLabel,
  overallStatusAlertVariant,
  overallStatusLabel,
  sortComponents,
  StatusPageLoadError,
  StatusPageNotFoundError,
  type PublicStatusPagePayload,
} from '../lib/status-page'

function StatusPageLoading(): ReactElement {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background text-muted-foreground">
      {t('statusPage.loading')}
    </div>
  )
}

function StatusPageMessage({ title, description }: { title: string; description?: string }): ReactElement {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Alert className="max-w-lg" variant="error">
        <AlertCircleIcon />
        <AlertTitle>{title}</AlertTitle>
        {description ? <AlertDescription>{description}</AlertDescription> : null}
      </Alert>
    </div>
  )
}

function IncidentCard({
  incident,
  componentNamesById,
  showResolvedAt = false,
}: {
  incident: PublicStatusPagePayload['incidents'][number]
  componentNamesById: Map<string, string>
  showResolvedAt?: boolean
}): ReactElement {
  const affectedNames = incident.affectedComponentIds
    .map((componentId) => componentNamesById.get(componentId))
    .filter((name): name is string => Boolean(name))

  return (
    <Frame>
      <FramePanel className="space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <h3 className="text-base font-semibold text-foreground">{incident.title}</h3>
            <p className="text-sm text-muted-foreground">{formatDateTime(incident.createdAt)}</p>
            {showResolvedAt && incident.resolvedAt ? (
              <p className="text-sm text-muted-foreground">
                {t('statusPage.incidents.resolvedAt', { date: formatDateTime(incident.resolvedAt) })}
              </p>
            ) : null}
          </div>
          <Badge variant={incidentStatusBadgeVariant(incident.status)}>
            {t(incidentStatusLabel(incident.status))}
          </Badge>
        </div>

        {affectedNames.length > 0 ? (
          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">{t('statusPage.incidents.affected')}</p>
            <div className="flex flex-wrap gap-2">
              {affectedNames.map((name) => (
                <Badge key={name} variant="outline">
                  {name}
                </Badge>
              ))}
            </div>
          </div>
        ) : null}

        {incident.updates.length > 0 ? (
          <div className="space-y-3">
            <p className="text-sm font-medium text-foreground">{t('statusPage.incidents.updates')}</p>
            <ol className="space-y-3">
              {incident.updates.map((update) => (
                <li
                  className="rounded-lg border border-border bg-muted/30 p-3"
                  key={update.id}
                >
                  <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                    <Badge variant={incidentStatusBadgeVariant(update.status)}>
                      {t(incidentStatusLabel(update.status))}
                    </Badge>
                    <time className="text-xs text-muted-foreground">
                      {formatDateTime(update.createdAt)}
                    </time>
                  </div>
                  <p className="whitespace-pre-wrap text-sm text-foreground">{update.body}</p>
                </li>
              ))}
            </ol>
          </div>
        ) : null}
      </FramePanel>
    </Frame>
  )
}

export function StatusPageRoute(): ReactElement {
  const { slug = '' } = useParams()
  const normalizedSlug = slug.trim().toLowerCase()
  const [payload, setPayload] = useState<PublicStatusPagePayload | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const [error, setError] = useState(false)

  const loadStatusPage = useCallback(async (): Promise<void> => {
    if (!normalizedSlug) {
      setNotFound(true)
      setLoading(false)
      return
    }

    try {
      const nextPayload = await fetchPublicStatusPage(
        normalizedSlug,
        statusPageConfig.apiPublicUrl,
      )
      setPayload(nextPayload)
      setNotFound(false)
      setError(false)
    } catch (loadError) {
      setPayload(null)
      if (loadError instanceof StatusPageNotFoundError) {
        setNotFound(true)
        setError(false)
      } else if (loadError instanceof StatusPageLoadError) {
        setError(true)
        setNotFound(false)
      } else {
        setError(true)
        setNotFound(false)
      }
    } finally {
      setLoading(false)
    }
  }, [normalizedSlug])

  useEffect(() => {
    setLoading(true)
    void loadStatusPage()
  }, [loadStatusPage])

  useEffect(() => {
    if (!normalizedSlug) {
      return undefined
    }

    const intervalId = window.setInterval(() => {
      void loadStatusPage()
    }, statusPageConfig.pollIntervalMs)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [loadStatusPage, normalizedSlug])

  const sortedComponents = useMemo(
    () => (payload ? sortComponents(payload.components) : []),
    [payload],
  )

  const componentNamesById = useMemo(() => {
    const map = new Map<string, string>()
    for (const component of sortedComponents) {
      map.set(component.id, component.name)
    }
    return map
  }, [sortedComponents])

  if (loading) {
    return <StatusPageLoading />
  }

  if (notFound) {
    return <StatusPageMessage title={t('statusPage.notFound')} />
  }

  if (error || !payload) {
    return <StatusPageMessage title={t('statusPage.error.load')} />
  }

  return (
    <div className="min-h-screen bg-background">
      <main className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-4 py-10 sm:px-6">
        <header className="space-y-2">
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">{payload.title}</h1>
        </header>

        <Alert variant={overallStatusAlertVariant(payload.overallStatus)}>
          <AlertTitle>{t('statusPage.overall.title')}</AlertTitle>
          <AlertDescription>{t(overallStatusLabel(payload.overallStatus))}</AlertDescription>
        </Alert>

        <section aria-labelledby="components-heading" className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground" id="components-heading">
            {t('statusPage.components.title')}
          </h2>

          {sortedComponents.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('statusPage.components.empty')}</p>
          ) : (
            <div className="grid gap-3">
              {sortedComponents.map((component) => (
                <Frame key={component.id}>
                  <FramePanel>
                    <FrameHeader className="flex-row items-start justify-between gap-3 px-0 py-0">
                      <div className="space-y-1">
                        <FrameTitle className="text-base">{component.name}</FrameTitle>
                        {component.description ? (
                          <FrameDescription>{component.description}</FrameDescription>
                        ) : null}
                      </div>
                      <ComponentStatusBadge status={component.status} />
                    </FrameHeader>
                  </FramePanel>
                </Frame>
              ))}
            </div>
          )}
        </section>

        <section aria-labelledby="incidents-heading" className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground" id="incidents-heading">
            {t('statusPage.incidents.title')}
          </h2>

          {payload.incidents.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('statusPage.incidents.empty')}</p>
          ) : (
            <div className="grid gap-4">
              {payload.incidents.map((incident) => (
                <IncidentCard
                  componentNamesById={componentNamesById}
                  incident={incident}
                  key={incident.id}
                />
              ))}
            </div>
          )}
        </section>

        <section aria-labelledby="resolved-incidents-heading" className="space-y-4">
          <Collapsible>
            <CollapsibleTrigger className="flex w-full items-center justify-between gap-3 text-left">
              <h2 className="text-lg font-semibold text-foreground" id="resolved-incidents-heading">
                {t('statusPage.incidents.resolved.title')}
              </h2>
              <div className="flex items-center gap-2">
                {payload.resolvedIncidents.length > 0 ? (
                  <span className="text-sm text-muted-foreground">
                    {t('statusPage.incidents.resolved.count', {
                      count: String(payload.resolvedIncidents.length),
                    })}
                  </span>
                ) : null}
                <ChevronDownIcon className="size-4 shrink-0 text-muted-foreground transition-transform in-data-panel-open:rotate-180" />
              </div>
            </CollapsibleTrigger>

            <CollapsiblePanel className="pt-4">
              {payload.resolvedIncidents.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {t('statusPage.incidents.resolved.empty')}
                </p>
              ) : (
                <div className="grid gap-4">
                  {payload.resolvedIncidents.map((incident) => (
                    <IncidentCard
                      componentNamesById={componentNamesById}
                      incident={incident}
                      key={incident.id}
                      showResolvedAt
                    />
                  ))}
                </div>
              )}
            </CollapsiblePanel>
          </Collapsible>
        </section>

        <SubscribeForm apiPublicUrl={statusPageConfig.apiPublicUrl} slug={normalizedSlug} />
      </main>
    </div>
  )
}
