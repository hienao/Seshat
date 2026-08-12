import { describe, expect, it } from 'vitest'
import { resources } from './resources'

function keys(value: object, prefix = ''): string[] {
  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return child && typeof child === 'object' ? keys(child, path) : [path]
  })
}

function leaves(value: object, prefix = ''): Record<string, string> {
  return Object.fromEntries(Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return child && typeof child === 'object' ? Object.entries(leaves(child, path)) : [[path, String(child)]]
  }))
}

function variables(value: string) {
  return [...value.matchAll(/{{\s*([^},\s]+).*?}}/g)].map((match) => match[1]).sort()
}

describe('translation resources', () => {
  it('keeps English and Chinese translation keys in sync', () => {
    expect(keys(resources.en.translation).sort()).toEqual(keys(resources['zh-CN'].translation).sort())
  })

  it('keeps interpolation variables in sync across languages', () => {
    const english = leaves(resources.en.translation)
    const chinese = leaves(resources['zh-CN'].translation)
    for (const key of Object.keys(english)) expect(variables(chinese[key])).toEqual(variables(english[key]))
  })

  it('keeps the required TMDB notice unchanged in both languages', () => {
    const notice = 'This product uses the TMDB API but is not endorsed or certified by TMDB.'
    expect(resources.en.translation.admin.tmdbNotice).toBe(notice)
    expect(resources['zh-CN'].translation.admin.tmdbNotice).toBe(notice)
  })
})
