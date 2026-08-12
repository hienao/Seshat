import { Input } from '@appica/ui-react/input'
import { Eye, EyeOff } from '@appica/icons-react'
import { useState, type ComponentProps, type CSSProperties } from 'react'
import { useTranslation } from 'react-i18next'

type SecretInputProps = Omit<ComponentProps<typeof Input>, 'type' | 'endSlot'> & {
  revealLabel: string
}

export function SecretInput({ revealLabel, value, disabled, ...props }: SecretInputProps) {
  const [visible, setVisible] = useState(false)
  const { t } = useTranslation()
  const hasValue = String(value ?? '').length > 0

  return (
    <Input
      {...props}
      type={visible ? 'text' : 'password'}
      value={value}
      disabled={disabled}
      endSlot={hasValue && (
        <button
          type="button"
          disabled={disabled}
          className="rounded p-1 text-neutral-500 hover:text-neutral-900 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:text-white"
          aria-label={`${t(visible ? 'common.actions.hide' : 'common.actions.show')} ${revealLabel}`}
          title={`${t(visible ? 'common.actions.hide' : 'common.actions.show')} ${revealLabel}`}
          onClick={() => setVisible((current) => !current)}
        >
          {visible ? <EyeOff size={17} /> : <Eye size={17} />}
        </button>
      )}
    />
  )
}

type SecretTextareaProps = ComponentProps<'textarea'> & {
  revealLabel: string
}

export function SecretTextarea({ revealLabel, value, disabled, className = '', style, ...props }: SecretTextareaProps) {
  const [visible, setVisible] = useState(false)
  const { t } = useTranslation()
  const hasValue = String(value ?? '').length > 0
  const concealedStyle = visible ? style : ({ ...style, WebkitTextSecurity: 'disc' } as CSSProperties)

  return (
    <div className="relative">
      <textarea {...props} value={value} disabled={disabled} className={`${className} pr-10`} style={concealedStyle} />
      {hasValue && (
        <button
          type="button"
          disabled={disabled}
          className="absolute right-2 top-2 rounded bg-white/90 p-1 text-neutral-500 hover:text-neutral-900 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-neutral-900/90 dark:hover:text-white"
          aria-label={`${t(visible ? 'common.actions.hide' : 'common.actions.show')} ${revealLabel}`}
          title={`${t(visible ? 'common.actions.hide' : 'common.actions.show')} ${revealLabel}`}
          onClick={() => setVisible((current) => !current)}
        >
          {visible ? <EyeOff size={17} /> : <Eye size={17} />}
        </button>
      )}
    </div>
  )
}
