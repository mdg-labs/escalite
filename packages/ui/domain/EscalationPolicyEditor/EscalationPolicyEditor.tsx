'use client'

import {
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ChevronDownIcon, GripVerticalIcon, PlusIcon, TrashIcon } from 'lucide-react'
import { useMemo, useState, type ReactElement } from 'react'

import { Button } from '../../primitives/button'
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from '../../primitives/collapsible'
import { Frame, FrameDescription, FramePanel, FrameTitle } from '../../primitives/frame'
import { Group } from '../../primitives/group'
import { Input } from '../../primitives/input'
import {
  NumberField,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
} from '../../primitives/number-field'
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from '../../primitives/select'
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
} from '../../primitives/alert-dialog'
import { addEscalationStep, normalizeStepOrders, removeEscalationStep } from './reorder'
import {
  createEmptyTarget,
  type EscalationEditorOptions,
  type EscalationEditorPolicy,
  type EscalationEditorStep,
  type EscalationPolicySavePayload,
  type EscalationStepTarget,
  type EscalationTargetType,
} from './types'
import { canSaveEscalationPolicy, validateEscalationPolicySteps } from './validation'

const TARGET_TYPE_LABELS: Record<EscalationTargetType, string> = {
  user: 'User',
  rotation: 'Rotation schedule',
  webhook: 'Webhook URL',
}

export type EscalationPolicyEditorProps = {
  policy: EscalationEditorPolicy
  options: EscalationEditorOptions
  saving?: boolean
  onChange: (policy: EscalationEditorPolicy) => void
  onSave: (payload: EscalationPolicySavePayload) => void | Promise<void>
}

function SortableStepCard({
  step,
  index,
  options,
  validationMessages,
  onChange,
  onRemove,
}: {
  step: EscalationEditorStep
  index: number
  options: EscalationEditorOptions
  validationMessages: string[]
  onChange: (step: EscalationEditorStep) => void
  onRemove: () => void
}): ReactElement {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: step.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  function updateTarget(targetId: string, patch: Partial<EscalationStepTarget>): void {
    onChange({
      ...step,
      targets: step.targets.map((target) =>
        target.id === targetId ? { ...target, ...patch } : target,
      ),
    })
  }

  function removeTarget(targetId: string): void {
    onChange({
      ...step,
      targets: step.targets.filter((target) => target.id !== targetId),
    })
  }

  function addTarget(): void {
    onChange({
      ...step,
      targets: [...step.targets, createEmptyTarget()],
    })
  }

  return (
    <div ref={setNodeRef} style={style} className={isDragging ? 'opacity-70' : undefined}>
      <Frame>
        <FramePanel className="p-0">
          <Collapsible defaultOpen>
            <div className="flex items-start gap-2 border-b border-border px-4 py-3">
              <button
                aria-label={`Reorder step ${index + 1}`}
                className="mt-0.5 inline-flex size-8 shrink-0 cursor-grab items-center justify-center rounded-md text-muted-foreground hover:bg-accent active:cursor-grabbing"
                type="button"
                {...attributes}
                {...listeners}
              >
                <GripVerticalIcon className="size-4" />
              </button>
              <div className="min-w-0 flex-1">
                <CollapsibleTrigger className="flex w-full items-center justify-between gap-3 text-left">
                  <div>
                    <FrameTitle>Step {index + 1}</FrameTitle>
                    <FrameDescription>
                      {index === 0
                        ? 'Notifies immediately when an alert triggers.'
                        : `Waits ${step.delayMinutes} minutes before escalating.`}
                    </FrameDescription>
                  </div>
                  <ChevronDownIcon className="size-4 shrink-0 text-muted-foreground transition-transform in-data-panel-open:rotate-180" />
                </CollapsibleTrigger>
              </div>
              <Button
                aria-label={`Delete step ${index + 1}`}
                onClick={onRemove}
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <TrashIcon />
              </Button>
            </div>
            <CollapsiblePanel>
              <div className="space-y-4 px-4 py-4">
                {index > 0 ? (
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">Delay (minutes)</label>
                    <NumberField
                      min={0}
                      onValueChange={(value) =>
                        onChange({
                          ...step,
                          delayMinutes: value ?? 0,
                        })
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
                ) : null}

                <div className="space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="text-sm font-medium text-foreground">Targets</p>
                    <Button onClick={addTarget} size="sm" type="button" variant="outline">
                      <PlusIcon />
                      Add target
                    </Button>
                  </div>

                  {step.targets.length === 0 ? (
                    <p className="text-sm text-destructive-foreground" role="alert">
                      Each step needs at least one target.
                    </p>
                  ) : null}

                  {step.targets.map((target) => (
                    <div
                      key={target.id}
                      className="space-y-3 rounded-lg border border-border bg-muted/20 p-3"
                    >
                      <Group className="justify-between">
                        <div className="min-w-0 flex-1 space-y-2">
                          <label className="text-sm font-medium text-foreground">Target type</label>
                          <Select
                            onValueChange={(value) =>
                              updateTarget(target.id, {
                                targetType: value as EscalationTargetType,
                                userId: undefined,
                                scheduleId: undefined,
                                webhookUrl: undefined,
                              })
                            }
                            value={target.targetType}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectPopup>
                              {(Object.keys(TARGET_TYPE_LABELS) as EscalationTargetType[]).map(
                                (targetType) => (
                                  <SelectItem key={targetType} value={targetType}>
                                    {TARGET_TYPE_LABELS[targetType]}
                                  </SelectItem>
                                ),
                              )}
                            </SelectPopup>
                          </Select>
                        </div>
                        <Button
                          aria-label="Remove target"
                          onClick={() => removeTarget(target.id)}
                          size="icon-sm"
                          type="button"
                          variant="ghost"
                        >
                          <TrashIcon />
                        </Button>
                      </Group>

                      {target.targetType === 'user' ? (
                        <div className="space-y-2">
                          <label className="text-sm font-medium text-foreground">User</label>
                          <Select
                            onValueChange={(value) =>
                              updateTarget(target.id, { userId: value ?? undefined })
                            }
                            value={target.userId ?? undefined}
                          >
                            <SelectTrigger aria-invalid={!target.userId}>
                              <SelectValue placeholder="Select a user" />
                            </SelectTrigger>
                            <SelectPopup>
                              {options.users.map((user) => (
                                <SelectItem key={user.id} value={user.id}>
                                  {user.label}
                                </SelectItem>
                              ))}
                            </SelectPopup>
                          </Select>
                        </div>
                      ) : null}

                      {target.targetType === 'rotation' ? (
                        <div className="space-y-2">
                          <label className="text-sm font-medium text-foreground">Schedule</label>
                          <Select
                            onValueChange={(value) =>
                              updateTarget(target.id, { scheduleId: value ?? undefined })
                            }
                            value={target.scheduleId ?? undefined}
                          >
                            <SelectTrigger aria-invalid={!target.scheduleId}>
                              <SelectValue placeholder="Select a schedule" />
                            </SelectTrigger>
                            <SelectPopup>
                              {options.schedules.map((schedule) => (
                                <SelectItem key={schedule.id} value={schedule.id}>
                                  {schedule.label}
                                </SelectItem>
                              ))}
                            </SelectPopup>
                          </Select>
                        </div>
                      ) : null}

                      {target.targetType === 'webhook' ? (
                        <div className="space-y-2">
                          <label className="text-sm font-medium text-foreground">Webhook URL</label>
                          <Input
                            aria-invalid={!target.webhookUrl?.trim()}
                            onChange={(event) =>
                              updateTarget(target.id, { webhookUrl: event.target.value })
                            }
                            placeholder="https://example.com/hooks/escalite"
                            value={target.webhookUrl ?? ''}
                          />
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>

                {validationMessages.length > 0 ? (
                  <div className="space-y-1" role="alert">
                    {validationMessages.map((message) => (
                      <p key={message} className="text-sm text-destructive-foreground">
                        {message}
                      </p>
                    ))}
                  </div>
                ) : null}
              </div>
            </CollapsiblePanel>
          </Collapsible>
        </FramePanel>
      </Frame>
    </div>
  )
}

export function EscalationPolicyEditor({
  policy,
  options,
  saving = false,
  onChange,
  onSave,
}: EscalationPolicyEditorProps): ReactElement {
  const [deleteStepId, setDeleteStepId] = useState<string | null>(null)
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  )

  const validationIssues = useMemo(
    () => validateEscalationPolicySteps(policy.steps),
    [policy.steps],
  )

  const issuesByStepId = useMemo(() => {
    const map = new Map<string, string[]>()
    for (const issue of validationIssues) {
      if (!issue.stepId) {
        continue
      }
      const current = map.get(issue.stepId) ?? []
      current.push(issue.message)
      map.set(issue.stepId, current)
    }
    return map
  }, [validationIssues])

  const globalIssues = validationIssues.filter((issue) => !issue.stepId)

  function updateSteps(steps: EscalationEditorStep[]): void {
    onChange({
      ...policy,
      steps: normalizeStepOrders(steps),
    })
  }

  function handleDragEnd(event: DragEndEvent): void {
    const { active, over } = event
    if (!over || active.id === over.id) {
      return
    }

    const oldIndex = policy.steps.findIndex((step) => step.id === active.id)
    const newIndex = policy.steps.findIndex((step) => step.id === over.id)
    if (oldIndex < 0 || newIndex < 0) {
      return
    }

    updateSteps(arrayMove(policy.steps, oldIndex, newIndex))
  }

  async function handleSave(): Promise<void> {
    if (!canSaveEscalationPolicy(policy.steps) || !policy.name.trim()) {
      return
    }

    await onSave({
      name: policy.name.trim(),
      steps: policy.steps.map((step, index) => ({
        stepOrder: index + 1,
        delayMinutes: index === 0 ? 0 : step.delayMinutes,
        repeatLastStep: step.repeatLastStep,
        maxRepeats: step.maxRepeats ?? undefined,
      })),
    })
  }

  const saveDisabled =
    saving || !policy.name.trim() || !canSaveEscalationPolicy(policy.steps)

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground" htmlFor="escalation-policy-name">
          Policy name
        </label>
        <Input
          id="escalation-policy-name"
          onChange={(event) => onChange({ ...policy, name: event.target.value })}
          placeholder="Default escalation"
          value={policy.name}
        />
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-foreground">Steps</h2>
            <p className="text-sm text-muted-foreground">
              Drag to reorder. Step order is saved when you update the policy.
            </p>
          </div>
          <Button
            onClick={() => updateSteps(addEscalationStep(policy.steps))}
            size="sm"
            type="button"
            variant="outline"
          >
            <PlusIcon />
            Add step
          </Button>
        </div>

        <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd} sensors={sensors}>
          <SortableContext
            items={policy.steps.map((step) => step.id)}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-3">
              {policy.steps.map((step, index) => (
                <SortableStepCard
                  key={step.id}
                  index={index}
                  onChange={(nextStep) =>
                    updateSteps(
                      policy.steps.map((current) =>
                        current.id === nextStep.id ? nextStep : current,
                      ),
                    )
                  }
                  onRemove={() => setDeleteStepId(step.id)}
                  options={options}
                  step={step}
                  validationMessages={issuesByStepId.get(step.id) ?? []}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      </div>

      {globalIssues.length > 0 ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3" role="alert">
          {globalIssues.map((issue) => (
            <p key={issue.message} className="text-sm text-destructive-foreground">
              {issue.message}
            </p>
          ))}
        </div>
      ) : null}

      <Group className="justify-end">
        <Button disabled={saveDisabled} loading={saving} onClick={() => void handleSave()} type="button">
          Save policy
        </Button>
      </Group>

      <AlertDialog onOpenChange={(open) => !open && setDeleteStepId(null)} open={deleteStepId !== null}>
        <AlertDialogPopup>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete escalation step?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the step from the policy. You cannot delete the only remaining step.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </AlertDialogClose>
            <AlertDialogClose
              onClick={() => {
                if (!deleteStepId) {
                  return
                }
                updateSteps(removeEscalationStep(policy.steps, deleteStepId))
                setDeleteStepId(null)
              }}
              render={<Button type="button" variant="destructive" />}
            >
              Delete step
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogPopup>
      </AlertDialog>
    </div>
  )
}
