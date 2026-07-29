export type SelectOption = {
  label: string
  value: string
  disabled?: boolean
}

export function selectOptionsFromEntities(
  entities: ReadonlyArray<{ id: string; name: string }>,
): SelectOption[] {
  return entities.map((entity) => ({ label: entity.name, value: entity.id }))
}

export function selectOptionsFromLabeledEntities(
  entities: ReadonlyArray<{ id: string; label: string }>,
): SelectOption[] {
  return entities.map((entity) => ({ label: entity.label, value: entity.id }))
}

export function selectOptionsWithSentinel(
  sentinelLabel: string,
  sentinelValue: string,
  options: SelectOption[],
): SelectOption[] {
  return [{ label: sentinelLabel, value: sentinelValue }, ...options]
}
