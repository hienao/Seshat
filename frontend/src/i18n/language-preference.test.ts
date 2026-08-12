import { describe, expect, it } from 'vitest'
import { browserLanguage, readLanguagePreference, resolveLanguage } from './language-preference'

describe('language preference', () => {
  it('maps every Chinese browser locale to Simplified Chinese', () => {
    expect(browserLanguage(['zh-HK'])).toBe('zh-CN')
    expect(browserLanguage(['fr-FR', 'zh-TW'])).toBe('zh-CN')
  })

  it('defaults non-Chinese and unknown browser locales to English', () => {
    expect(browserLanguage(['en-US'])).toBe('en')
    expect(browserLanguage([])).toBe('en')
  })

  it('uses a valid persisted preference and rejects invalid values', () => {
    const storage = { getItem: () => 'en' } as unknown as Storage
    const invalidStorage = { getItem: () => 'fr' } as unknown as Storage
    expect(readLanguagePreference(storage)).toBe('en')
    expect(readLanguagePreference(invalidStorage)).toBe('auto')
    expect(resolveLanguage('zh-CN')).toBe('zh-CN')
  })
})
