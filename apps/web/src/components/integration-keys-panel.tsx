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
import { formatDateTime, formatGraphQLError } from '../lib/format'
import { buildInboundWebhookURL } from '../lib/integration-presets'
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
        Revoked
      </Badge>
    )
  }

  return (
    <Badge variant="success">
      <CircleCheckIcon />
      Active
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
      setError(formatGraphQLError(result.error.message))
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
      setError(formatGraphQLError(result.error.message))
      return
    }

    const rotated = result.data?.rotateIntegrationKey
    if (!rotated?.token) {
      setError('Rotated key was created but the token was not returned.')
      reexecuteQuery({ requestPolicy: 'network-only' })
      return
    }

    setRevealSecret({
      title: 'Integration key rotated',
      description:
        'The previous key is revoked. Save the new webhook URL below — the full token is shown once.',
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
      title: `${payload.presetLabel} integration key created`,
      description:
        'Save the webhook URL below. After you close this dialog, only the token prefix remains visible.',
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
            Service
          </label>
          <Combobox onValueChange={handleServiceSelect} value={serviceId || null}>
            <ComboboxInput
              id="integration-service-picker"
              placeholder="Select a service"
              showClear
            />
            <ComboboxPopup>
              <ComboboxList>
                <ComboboxEmpty>
                  {servicesFetching ? 'Loading services…' : 'No services found.'}
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
          <AlertTitle>Action failed</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {loadedServiceId ? (
        <section className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-foreground">Integration keys</h2>
              <p className="text-sm text-muted-foreground">
                Prefix-only display after creation. Revoked keys stop accepting webhooks immediately.
              </p>
            </div>
            <Dialog onOpenChange={setCreateOpen} open={createOpen}>
              <DialogTrigger render={<Button type="button" />}>Create key</DialogTrigger>
              <DialogPopup>
                <DialogHeader>
                  <DialogTitle>Create integration key</DialogTitle>
                  <DialogDescription>
                    Choose a preset to create an inbound integration key with pre-filled field mapping.
                  </DialogDescription>
                </DialogHeader>
                <DialogPanel>
                  <IntegrationPicker
                    onCreated={handleCreated}
                    serviceId={loadedServiceId}
                  />
                </DialogPanel>
                <DialogFooter variant="bare">
                  <DialogClose render={<Button type="button" variant="ghost" />}>Close</DialogClose>
                </DialogFooter>
              </DialogPopup>
            </Dialog>
          </div>

          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead>Plugin</TableHead>
                <TableHead>Token prefix</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {fetching ? (
                <TableRow>
                  <TableCell className="text-muted-foreground" colSpan={5}>
                    Loading integration keys…
                  </TableCell>
                </TableRow>
              ) : keys.length === 0 ? (
                <TableRow>
                  <TableCell className="text-muted-foreground" colSpan={5}>
                    No integration keys for this service yet.
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
                            Rotate
                          </Button>
                          <AlertDialog>
                            <AlertDialogTrigger
                              render={<Button size="sm" type="button" variant="destructive-outline" />}
                            >
                              Revoke
                            </AlertDialogTrigger>
                            <AlertDialogPopup>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Revoke integration key?</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Webhook requests using prefix{' '}
                                  <span className="font-mono text-foreground">{key.tokenPrefix}</span>{' '}
                                  will return 404 immediately after revocation.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                                  Cancel
                                </AlertDialogClose>
                                <AlertDialogClose
                                  onClick={() => void handleRevoke(key)}
                                  render={<Button type="button" variant="destructive" />}
                                >
                                  Revoke key
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
                  <AlertTitle>Shown once</AlertTitle>
                  <AlertDescription>
                    Token prefix <span className="font-mono text-foreground">{revealSecret.tokenPrefix}</span>{' '}
                    will remain visible in the key list after you close this dialog.
                  </AlertDescription>
                </Alert>
                <div className="space-y-2">
                  <p className="text-sm font-medium text-foreground">Webhook URL</p>
                  <code className="block overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                    {revealSecret.webhookURL}
                  </code>
                  <CopyButton label="Copy URL" value={revealSecret.webhookURL} />
                </div>
              </DialogPanel>
              <DialogFooter>
                <DialogClose render={<Button type="button" />}>I saved the URL</DialogClose>
              </DialogFooter>
            </>
          ) : null}
        </DialogPopup>
      </Dialog>
    </div>
  )
}
