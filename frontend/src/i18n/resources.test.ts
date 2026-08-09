import { describe, expect, it } from 'vitest'
import { resources } from './resources'

function keys(value: object, prefix = ''): string[] {
  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return child && typeof child === 'object' ? keys(child, path) : [path]
  })
}

describe('translation resources', () => {
  it('keeps English and Chinese translation keys in sync', () => {
    expect(keys(resources.en.translation).sort()).toEqual(keys(resources['zh-CN'].translation).sort())
  })

  it('keeps the required TMDB notice unchanged in both languages', () => {
    const notice = 'This product uses the TMDB API but is not endorsed or certified by TMDB.'
    expect(resources.en.translation.admin.tmdbNotice).toBe(notice)
    expect(resources['zh-CN'].translation.admin.tmdbNotice).toBe(notice)
  })
})
