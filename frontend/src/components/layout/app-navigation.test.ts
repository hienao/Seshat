import { describe, expect, it } from 'vitest'
import { getCurrentPageTitleKey, getNavigationSections } from './app-navigation'

describe('app navigation', () => {
  it('only exposes administrator navigation to administrators', () => {
    expect(getNavigationSections(false).map((section) => section.labelKey)).toEqual(['navigation.primary'])
    expect(getNavigationSections(true).map((section) => section.labelKey)).toEqual(['navigation.primary', 'navigation.administration'])
    expect(getNavigationSections(true)[1].items.map((item) => item.to)).toEqual(['/admin', '/admin/logs', '/admin/application-logs'])
    expect(getNavigationSections(false)[0].items.map((item) => item.to)).toContain('/notification-channels')
  })

  it('resolves nested routes to the correct page title', () => {
    expect(getCurrentPageTitleKey('/events/42')).toBe('navigation.eventDetail')
    expect(getCurrentPageTitleKey('/admin/logs')).toBe('navigation.apiLogs')
    expect(getCurrentPageTitleKey('/admin/application-logs')).toBe('navigation.applicationLogs')
    expect(getCurrentPageTitleKey('/notification-channels')).toBe('navigation.notificationChannels')
    expect(getCurrentPageTitleKey('/admin')).toBe('navigation.system')
    expect(getCurrentPageTitleKey('/missing')).toBe('navigation.notFound')
  })
})
