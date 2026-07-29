import { useEffect, useMemo, useState, type ReactElement } from 'react'
import {
  AlertPriority,
  useDeleteNotificationRuleMutation,
  useNotificationChannelsQuery,
  useNotificationRulesQuery,
  useSaveNotificationRuleMutation,
  type NotificationChannelsQuery,
  type NotificationRulesQuery,
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
  Button,
  NumberField,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  Tabs,
  TabsList,
  TabsPanel,
  TabsTab,
} from '@escalite/ui'
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ListOrderedIcon,
  PlusIcon,
  TrashIcon,
} from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t, type MessageKey } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

type ChannelDefinition = NotificationChannelsQuery['notificationChannels'][number]
type UserNotificationRule = NotificationRulesQuery['notificationRules'][number]

type EditableStep = {
  id: string
  channel: string
  delayMinutes: number
}

const CHANNEL_LABEL_KEYS: Record<string, MessageKey> = {
  email: 'settings.contactMethods.channel.email',
  push: 'settings.contactMethods.channel.push',
  'slack-dm': 'settings.contactMethods.channel.slackDm',
  webhook: 'settings.contactMethods.channel.webhook',
}

function channelLabel(name: string): string {
  const key = CHANNEL_LABEL_KEYS[name]
  return key ? t(key) : name
}

function createStepId(): string {
  return `step-${crypto.randomUUID()}`
}

function createEmptyStep(channel = ''): EditableStep {
  return {
    id: createStepId(),
    channel,
    delayMinutes: 0,
  }
}

function stepsFromRule(rule: UserNotificationRule | undefined): EditableStep[] {
  if (!rule || rule.steps.length === 0) {
    return []
  }

  return rule.steps.map((step, index) => ({
    id: createStepId(),
    channel: step.channel,
    delayMinutes: index === 0 ? 0 : step.delayMinutes,
  }))
}

function normalizeSteps(steps: EditableStep[]): EditableStep[] {
  return steps.map((step, index) => ({
    ...step,
    delayMinutes: index === 0 ? 0 : step.delayMinutes,
  }))
}

function validateSteps(steps: EditableStep[]): string | null {
  if (steps.length === 0) {
    return t('settings.notificationRules.validation.minSteps')
  }

  const seen = new Set<string>()
  for (const step of steps) {
    const channel = step.channel.trim()
    if (!channel) {
      return t('settings.notificationRules.validation.required')
    }
    if (seen.has(channel)) {
      return t('settings.notificationRules.validation.duplicateChannel')
    }
    if (step.delayMinutes < 0) {
      return t('settings.notificationRules.validation.delay')
    }
    seen.add(channel)
  }

  return null
}

function PriorityRuleEditor({
  priority,
  channels,
  existingRule,
  onChanged,
}: {
  priority: AlertPriority
  channels: ChannelDefinition[]
  existingRule: UserNotificationRule | undefined
  onChanged: () => void
}): ReactElement {
  const [, saveNotificationRule] = useSaveNotificationRuleMutation()
  const [, deleteNotificationRule] = useDeleteNotificationRuleMutation()
  const [steps, setSteps] = useState<EditableStep[]>(() => stepsFromRule(existingRule))
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const defaultChannel = channels[0]?.name ?? ''

  useEffect(() => {
    setSteps(stepsFromRule(existingRule))
    setActionError(null)
  }, [existingRule])

  function updateStep(stepId: string, patch: Partial<EditableStep>): void {
    setSteps((current) =>
      normalizeSteps(
        current.map((step) => (step.id === stepId ? { ...step, ...patch } : step)),
      ),
    )
    setActionError(null)
  }

  function addStep(): void {
    setSteps((current) =>
      normalizeSteps([
        ...current,
        createEmptyStep(
          channels.find((channel) => !current.some((step) => step.channel === channel.name))
            ?.name ?? defaultChannel,
        ),
      ]),
    )
    setActionError(null)
  }

  function removeStep(stepId: string): void {
    setSteps((current) => normalizeSteps(current.filter((step) => step.id !== stepId)))
    setActionError(null)
  }

  function moveStep(stepId: string, direction: -1 | 1): void {
    setSteps((current) => {
      const index = current.findIndex((step) => step.id === stepId)
      if (index === -1) {
        return current
      }

      const targetIndex = index + direction
      if (targetIndex < 0 || targetIndex >= current.length) {
        return current
      }

      const next = [...current]
      const [moved] = next.splice(index, 1)
      next.splice(targetIndex, 0, moved)
      return normalizeSteps(next)
    })
    setActionError(null)
  }

  async function handleSave(): Promise<void> {
    setActionError(null)

    const validationError = validateSteps(steps)
    if (validationError) {
      setActionError(validationError)
      return
    }

    setSaving(true)
    const result = await saveNotificationRule({
      input: {
        priority,
        steps: steps.map((step) => ({
          channel: step.channel,
          delayMinutes: step.delayMinutes,
        })),
      },
    })
    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'notificationRules.error.action')
      return
    }

    notifyMutationSuccess('notificationRules.toast.saved')
    onChanged()
  }

  async function handleDelete(): Promise<void> {
    setActionError(null)
    setDeleting(true)

    const result = await deleteNotificationRule({ priority })
    setDeleting(false)

    if (result.error) {
      showMutationError(result.error, 'notificationRules.error.action')
      return
    }

    setSteps([])
    onChanged()
  }

  const usedChannels = new Set(steps.map((step) => step.channel).filter(Boolean))

  return (
    <div className="space-y-4">
      {steps.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t('settings.notificationRules.empty')}</p>
      ) : (
        <ol className="space-y-3">
          {steps.map((step, index) => {
            const channelItems = channels.map((channel) => ({
              label: channelLabel(channel.name),
              value: channel.name,
              disabled: channel.name !== step.channel && usedChannels.has(channel.name),
            }))

            return (
            <li
              className="space-y-4 rounded-lg border border-border bg-muted/30 p-4"
              key={step.id}
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="text-sm font-medium text-foreground">
                    {t('settings.notificationRules.step.title', { step: String(index + 1) })}
                  </h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {index === 0
                      ? t('settings.notificationRules.step.delayFirst')
                      : t('settings.notificationRules.step.delayHelp')}
                  </p>
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    aria-label={t('settings.notificationRules.action.moveUp')}
                    disabled={index === 0}
                    onClick={() => moveStep(step.id, -1)}
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  >
                    <ArrowUpIcon />
                  </Button>
                  <Button
                    aria-label={t('settings.notificationRules.action.moveDown')}
                    disabled={index === steps.length - 1}
                    onClick={() => moveStep(step.id, 1)}
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  >
                    <ArrowDownIcon />
                  </Button>
                  <Button
                    aria-label={t('settings.notificationRules.action.removeStep')}
                    onClick={() => removeStep(step.id)}
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  >
                    <TrashIcon />
                  </Button>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <label
                    className="text-sm font-medium text-foreground"
                    htmlFor={`${priority}-${step.id}-channel`}
                  >
                    {t('settings.notificationRules.step.channel')}
                  </label>
                  <Select
                    items={channelItems}
                    onValueChange={(value) => updateStep(step.id, { channel: value ?? '' })}
                    value={step.channel || null}
                  >
                    <SelectTrigger id={`${priority}-${step.id}-channel`}>
                      <SelectValue placeholder={t('settings.notificationRules.step.channelPlaceholder')} />
                    </SelectTrigger>
                    <SelectPopup>
                      {channelItems.map((item) => (
                        <SelectItem disabled={item.disabled} key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectPopup>
                  </Select>
                </div>

                <div className="space-y-2">
                  <label
                    className="text-sm font-medium text-foreground"
                    htmlFor={`${priority}-${step.id}-delay`}
                  >
                    {t('settings.notificationRules.step.delay')}
                  </label>
                  <NumberField
                    disabled={index === 0}
                    id={`${priority}-${step.id}-delay`}
                    min={0}
                    onValueChange={(value) =>
                      updateStep(step.id, { delayMinutes: value ?? 0 })
                    }
                    value={step.delayMinutes}
                  >
                    <NumberFieldGroup>
                      <NumberFieldDecrement />
                      <NumberFieldInput />
                      <NumberFieldIncrement />
                    </NumberFieldGroup>
                  </NumberField>
                </div>
              </div>
            </li>
            )
          })}
        </ol>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <Button
          disabled={channels.length === 0 || steps.length >= channels.length}
          onClick={addStep}
          type="button"
          variant="outline"
        >
          <PlusIcon />
          {t('settings.notificationRules.action.addStep')}
        </Button>
        <Button disabled={saving || steps.length === 0} onClick={() => void handleSave()} type="button">
          {saving
            ? t('settings.notificationRules.action.saving')
            : t('settings.notificationRules.action.save')}
        </Button>
        {existingRule ? (
          <AlertDialog>
            <AlertDialogTrigger
              render={
                <Button
                  disabled={deleting}
                  type="button"
                  variant="destructive-outline"
                />
              }
            >
              {deleting
                ? t('settings.notificationRules.action.deleting')
                : t('settings.notificationRules.action.delete')}
            </AlertDialogTrigger>
            <AlertDialogPopup>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('settings.notificationRules.delete.title')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t('settings.notificationRules.delete.description')}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                  {t('settings.devices.action.cancel')}
                </AlertDialogClose>
                <AlertDialogClose
                  onClick={() => void handleDelete()}
                  render={<Button type="button" variant="destructive" />}
                >
                  {t('settings.notificationRules.delete.confirm')}
                </AlertDialogClose>
              </AlertDialogFooter>
            </AlertDialogPopup>
          </AlertDialog>
        ) : null}
      </div>

      {actionError ? (
        <Alert variant="error">
          <AlertDescription>{actionError}</AlertDescription>
        </Alert>
      ) : null}
    </div>
  )
}

export function NotificationRulesPanel(): ReactElement {
  const [{ data: channelsData, fetching: channelsFetching, error: channelsError }] =
    useNotificationChannelsQuery({
      requestPolicy: 'cache-first',
    })
  const [{ data: rulesData, fetching: rulesFetching, error: rulesError }, reexecuteRulesQuery] =
    useNotificationRulesQuery({
      requestPolicy: 'network-only',
    })

  const channels = channelsData?.notificationChannels ?? []
  const rules = useMemo(() => rulesData?.notificationRules ?? [], [rulesData?.notificationRules])

  const rulesByPriority = useMemo(
    () => ({
      [AlertPriority.High]: rules.find((rule) => rule.priority === AlertPriority.High),
      [AlertPriority.Low]: rules.find((rule) => rule.priority === AlertPriority.Low),
    }),
    [rules],
  )

  const loading = (channelsFetching && channels.length === 0) || (rulesFetching && rules.length === 0)
  const queryError = channelsError ?? rulesError

  function handleChanged(): void {
    reexecuteRulesQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <ListOrderedIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">
            {t('settings.notificationRules.title')}
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            {t('settings.notificationRules.description')}
          </p>
        </div>
      </div>

      {queryError ? (
        <Alert className="mt-6" variant="error">
          <AlertDescription>{formatGraphQLError(queryError.message)}</AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6">
        {loading ? (
          <p className="text-sm text-muted-foreground">{t('settings.notificationRules.loading')}</p>
        ) : (
          <Tabs defaultValue={AlertPriority.High}>
            <TabsList variant="underline">
              <TabsTab value={AlertPriority.High}>{t('settings.notificationRules.tab.high')}</TabsTab>
              <TabsTab value={AlertPriority.Low}>{t('settings.notificationRules.tab.low')}</TabsTab>
            </TabsList>

            <TabsPanel className="mt-6" value={AlertPriority.High}>
              <PriorityRuleEditor
                channels={channels}
                existingRule={rulesByPriority[AlertPriority.High]}
                onChanged={handleChanged}
                priority={AlertPriority.High}
              />
            </TabsPanel>

            <TabsPanel className="mt-6" value={AlertPriority.Low}>
              <PriorityRuleEditor
                channels={channels}
                existingRule={rulesByPriority[AlertPriority.Low]}
                onChanged={handleChanged}
                priority={AlertPriority.Low}
              />
            </TabsPanel>
          </Tabs>
        )}
      </div>
    </section>
  )
}
