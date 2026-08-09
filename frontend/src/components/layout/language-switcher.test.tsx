import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import { LANGUAGE_STORAGE_KEY } from '@/i18n/language-preference'
import { LanguageSwitcher } from './language-switcher'

describe('LanguageSwitcher', () => {
  it('persists a manual choice and applies it immediately', async () => {
    render(<LanguageSwitcher />)
    fireEvent.change(screen.getByRole('combobox', { name: '语言' }), { target: { value: 'en' } })
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe('en')
    expect(await screen.findByRole('combobox', { name: 'Language' })).toHaveValue('en')
    expect(i18n.resolvedLanguage).toBe('en')
    expect(document.documentElement.lang).toBe('en')
  })
})
