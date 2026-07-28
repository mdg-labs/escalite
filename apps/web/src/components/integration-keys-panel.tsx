import { useEffect, useMemo, useState, type ReactElement } from 'react'
import {
  useIntegrationKeysQuery,
  useRevokeIntegrationKeyMutation,
  useRotateIntegrationKeyMutation,
  useServicesQuery,
  type IntegrationKeysQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
  AlertDialogTrigger,
  AlertTitle,
  Badge,
  Button,
  Combobox,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxPopup,
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
  DialogTrigger,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { AlertTriangleIcon, CircleCheckIcon, InfoIcon } from 'lucide-react'

import { appConfig } from '../lib/config'
import { formatDateTime } from '../lib/format'
import { buildInboundWebhookURL } from '../lib/integration-presets'
import { t } from '../lib/i18n'
import { showMutationError } from '../lib/toast'
import { CopyButton } from './copy-button'
import { IntegrationPicker } from './integration-picker'

type IntegrationKeyRow = IntegrationKeysQuery['integrationKeys'][number]

type RevealSecret = {
  title: string
  description: string
  webhookURL: string
  tokenPrefix: string
}

type IntegrationKeysPanelProps = {
  serviceId?: string
  embedded?: boolean
}

function keyStatusBadge(key: IntegrationKeyRow): ReactElement {
  if (key.revokedAt) {
    return (
      <Badge variant="error">
        <AlertTriangleIcon />
        {t('integrationKeys.status.revoked')}
      </Badge>
    )
  }

  return (
    <Badge variant="success">
      <CircleCheckIcon />
      {t('integrationKeys.status.active')}
    </Badge>
  )
}

export function IntegrationKeysPanel({
  serviceId: initialServiceId = '',
  embedded = false,
}: IntegrationKeysPanelProps): ReactElement {
  const [serviceId, setServiceId] = useState(initialServiceId)
  const [loadedServiceId, setLoadedServiceId] = useState(embedded ? initialServiceId : '')
  const [createOpen, setCreateOpen] = useState(false)
  const [revealSecret, setRevealSecret] = useState<RevealSecret | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [actionKeyId, setActionKeyId] = useState<string | null>(null)

  const [{ data, fetching }, reexecuteQuery] = useIntegrationKeysQuery({
    pause: !loadedServiceId,
    variables: { serviceId: loadedServiceId },
    requestPolicy: 'network-only',
  })

  const [, revokeIntegrationKey] = useRevokeIntegrationKeyMutation()
  const [, rotateIntegrationKey] = useRotateIntegrationKeyMutation()
  const [{ data: servicesData, fetching: servicesFetching }] = useServicesQuery({
    pause: embedded,
    requestPolicy: 'cache-first',
  })

  const services = useMemo(() => servicesData?.services ?? [], [servicesData?.services])
  const keys = data?.integrationKeys ?? []

  useEffect(() => {
    if (embedded && initialServiceId) {
      setServiceId(initialServiceId)
      setLoadedServiceId(initialServiceId)
    }
  }, [embedded, initialServiceId])

  function handleServiceSelect(value: string | null): void {
    const next = value ?? ''
    setServiceId(next)
    setLoadedServiceId(next)
    setError(null)
  }

  async function handleRevoke(key: IntegrationKeyRow): Promise<void> {
    setError(null)
    setActionKeyId(key.id)
    const result = await revokeIntegrationKey({ id: key.id })
    setActionKeyId(null)

    if (result.error) {
      showMutationError(result.error, 'integrations.error.action')
      return
    }

    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  async function handleRotate(key: IntegrationKeyRow): Promise<void> {
    setError(null)
    setActionKeyId(key.id)
    const result = await rotateIntegrationKey({ id: key.id })
    setActionKeyId(null)

    if (result.error) {
      showMutationError(result.error, 'integrations.error.action')
      return
    }

    const rotated = result.data?.rotateIntegrationKey
    if (!rotated?.token) {
      showMutationError(t('integrationKeys.error.rotateNoToken'), 'integrations.error.action')
      reexecuteQuery({ requestPolicy: 'network-only' })
      return
    }

    setRevealSecret({
      title: t('integrationKeys.reveal.rotatedTitle'),
      description: t('integrationKeys.reveal.rotatedDescription'),
      tokenPrefix: rotated.tokenPrefix,
      webhookURL: buildInboundWebhookURL(appConfig.apiPublicUrl, rotated.pluginName, rotated.token),
    })
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  function handleCreated(payload: {
    pluginName: string
    token: string
    tokenPrefix: string
    presetLabel: string
  }): void {
    setCreateOpen(false)
    setRevealSecret({
      title: t('integrationKeys.reveal.createdTitle', { preset: payload.presetLabel }),
      description: t('integrationKeys.reveal.createdDescription'),
      tokenPrefix: payload.tokenPrefix,
      webhookURL: buildInboundWebhookURL(
        appConfig.apiPublicUrl,
        payload.pluginName,
        payload.token,
      ),
    })
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <div className="space-y-6">
      {!embedded ? (
        <div className="max-w-md space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="integration-service-picker">
            {t('integrationKeys.serviceLabel')}
          </label>
          <Combobox onValueChange={handleServiceSelect} value={serviceId || null}>
            <ComboboxInput
              id="integration-service-picker"
              placeholder={t('integrationKeys.servicePlaceholder')}
              showClear
            />
            <ComboboxPopup>
              <ComboboxList>
                <ComboboxEmpty>
                  {servicesFetching ? t('integrationKeys.loadingServices') : t('integrationKeys.noServices')}
                </ComboboxEmpty>
                {services.map((service) => (
                  <ComboboxItem key={service.id} value={service.id}>
                    {service.name}
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxPopup>
          </Combobox>
        </div>
      ) : null}

      {error ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertTitle>{t('common.actionFailed')}</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {loadedServiceId ? (
        <section className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-foreground">{t('integrationKeys.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('integrationKeys.description')}</p>
            </div>
            <Dialog onOpenChange={setCreateOpen} open={createOpen}>
              <DialogTrigger render={<Button type="button" />}>{t('integrationKeys.create')}</DialogTrigger>
              <DialogPopup>
                <DialogHeader>
                  <DialogTitle>{t('integrationKeys.createTitle')}</DialogTitle>
                  <DialogDescription>{t('integrationKeys.createDescription')}</DialogDescription>
                </DialogHeader>
                <DialogPanel>
                  <IntegrationPicker
                    onCreated={handleCreated}
                    serviceId={loadedServiceId}
                  />
                </DialogPanel>
                <DialogFooter variant="bare">
                  <DialogClose render={<Button type="button" variant="ghost" />}>{t('common.close')}</DialogClose>
                </DialogFooter>
              </DialogPopup>
            </Dialog>
          </div>

          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead>{t('integrationKeys.column.plugin')}</TableHead>
                <TableHead>{t('integrationKeys.column.tokenPrefix')}</TableHead>
                <TableHead>{t('common.status')}</TableHead>
                <TableHead>{t('common.created')}</TableHead>
                <TableHead className="text-right">{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {fetching ? (
                <TableRow>
                  <TableCell className="text-muted-foreground" colSpan={5}>
                    {t('integrationKeys.loading')}
                  </TableCell>
                </TableRow>
              ) : keys.length === 0 ? (
                <TableRow>
                  <TableCell className="text-muted-foreground" colSpan={5}>
                    {t('integrationKeys.empty')}
                  </TableCell>
                </TableRow>
              ) : (
                keys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium text-foreground">{key.pluginName}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-2 py-1 text-xs">{key.tokenPrefix}</code>
                    </TableCell>
                    <TableCell>{keyStatusBadge(key)}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDateTime(key.createdAt)}
                    </TableCell>
                    <TableCell className="text-right">
                      {key.revokedAt ? (
                        <span className="text-sm text-muted-foreground">—</span>
                      ) : (
                        <div className="flex justify-end gap-2">
                          <Button
                            disabled={actionKeyId === key.id}
                            onClick={() => void handleRotate(key)}
                            size="sm"
                            type="button"
                            variant="outline"
                          >
                            {t('integrationKeys.action.rotate')}
                          </Button>
                          <AlertDialog>
                            <AlertDialogTrigger
                              render={<Button size="sm" type="button" variant="destructive-outline" />}
                            >
                              {t('integrationKeys.action.revoke')}
                            </AlertDialogTrigger>
                            <AlertDialogPopup>
                              <AlertDialogHeader>
                                <AlertDialogTitle>{t('integrationKeys.revoke.title')}</AlertDialogTitle>
                                <AlertDialogDescription>
                                  {t('integrationKeys.revoke.description', { prefix: key.tokenPrefix })}
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                                  {t('common.cancel')}
                                </AlertDialogClose>
                                <AlertDialogClose
                                  onClick={() => void handleRevoke(key)}
                                  render={<Button type="button" variant="destructive" />}
                                >
                                  {t('integrationKeys.revoke.confirm')}
                                </AlertDialogClose>
                              </AlertDialogFooter>
                            </AlertDialogPopup>
                          </AlertDialog>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </section>
      ) : null}

      <Dialog onOpenChange={(open) => !open && setRevealSecret(null)} open={revealSecret !== null}>
        <DialogPopup>
          {revealSecret ? (
            <>
              <DialogHeader>
                <DialogTitle>{revealSecret.title}</DialogTitle>
                <DialogDescription>{revealSecret.description}</DialogDescription>
              </DialogHeader>
              <DialogPanel className="space-y-4">
                <Alert variant="warning">
                  <InfoIcon />
                  <AlertTitle>{t('common.shownOnce')}</AlertTitle>
                  <AlertDescription>
                    {t('integrationKeys.reveal.shownOnceDescription', { prefix: revealSecret.tokenPrefix })}
                  </AlertDescription>
                </Alert>
                <div className="space-y-2">
                  <p className="text-sm font-medium text-foreground">{t('integrationKeys.reveal.webhookUrl')}</p>
                  <code className="block overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                    {revealSecret.webhookURL}
                  </code>
                  <CopyButton label={t('common.copyUrl')} value={revealSecret.webhookURL} />
                </div>
              </DialogPanel>
              <DialogFooter>
                <DialogClose render={<Button type="button" />}>{t('common.savedUrl')}</DialogClose>
              </DialogFooter>
            </>
          ) : null}
        </DialogPopup>
      </Dialog>
    </div>
  )
}
