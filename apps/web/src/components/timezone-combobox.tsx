import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import {
  Combobox,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxPopup,
} from '@escalite/ui'

import { filterIanaTimezones, isValidIanaTimezone, listIanaTimezones } from '../lib/timezones'
import { t } from '../lib/i18n'

type TimezoneComboboxProps = {
  id?: string
  value: string
  onValueChange: (timezone: string) => void
  disabled?: boolean
  invalid?: boolean
}

export function TimezoneCombobox({
  id,
  value,
  onValueChange,
  disabled = false,
  invalid = false,
}: TimezoneComboboxProps): ReactElement {
  const allTimezones = useMemo(() => listIanaTimezones(), [])
  const [inputValue, setInputValue] = useState(value)

  useEffect(() => {
    setInputValue(value)
  }, [value])

  const filteredTimezones = useMemo(
    () => filterIanaTimezones(allTimezones, inputValue),
    [allTimezones, inputValue],
  )

  const showInvalidHint = invalid && inputValue.trim() !== '' && !isValidIanaTimezone(inputValue)

  return (
    <div className="space-y-1">
      <Combobox
        disabled={disabled}
        inputValue={inputValue}
        onInputValueChange={(nextValue) => {
          setInputValue(nextValue)
          if (isValidIanaTimezone(nextValue)) {
            onValueChange(nextValue.trim())
          }
        }}
        onValueChange={(nextValue) => {
          if (!nextValue) {
            return
          }

          setInputValue(nextValue)
          onValueChange(nextValue)
        }}
        value={isValidIanaTimezone(value) ? value : null}
      >
        <ComboboxInput
          aria-invalid={showInvalidHint}
          id={id}
          placeholder={t('schedule.form.timezonePlaceholder')}
          showClear
        />
        <ComboboxPopup>
          <ComboboxList>
            <ComboboxEmpty>{t('schedule.form.timezoneEmpty')}</ComboboxEmpty>
            {filteredTimezones.map((timezone) => (
              <ComboboxItem key={timezone} value={timezone}>
                {timezone}
              </ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxPopup>
      </Combobox>
      {showInvalidHint ? (
        <p className="text-sm text-destructive-foreground" role="alert">
          {t('schedule.form.error.invalidTimezone')}
        </p>
      ) : null}
    </div>
  )
}
