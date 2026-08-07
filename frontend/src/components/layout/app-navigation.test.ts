import { describe, expect, it } from 'vitest'
import { getCurrentPageTitle, getNavigationSections } from './app-navigation'

describe('app navigation', () => {
  it('only exposes administrator navigation to administrators', () => {
    expect(getNavigationSections(false).map((section) => section.label)).toEqual(['主要功能'])
    expect(getNavigationSections(true).map((section) => section.label)).toEqual(['主要功能', '系统管理'])
    expect(getNavigationSections(true)[1].items.map((item) => item.to)).toEqual(['/admin', '/admin/logs', '/admin/application-logs'])
    expect(getNavigationSections(false)[0].items.map((item) => item.to)).toContain('/notification-channels')
  })

  it('resolves nested routes to the correct page title', () => {
    expect(getCurrentPageTitle('/events/42')).toBe('消息详情')
    expect(getCurrentPageTitle('/admin/logs')).toBe('接口日志')
    expect(getCurrentPageTitle('/admin/application-logs')).toBe('业务日志')
    expect(getCurrentPageTitle('/notification-channels')).toBe('推送渠道')
    expect(getCurrentPageTitle('/admin')).toBe('系统管理')
    expect(getCurrentPageTitle('/missing')).toBe('页面未找到')
  })
})
