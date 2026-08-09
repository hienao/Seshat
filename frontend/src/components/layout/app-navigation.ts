import { Activity, Bell, FileText, Home, Plug, ShieldCheck, User } from '@appica/icons-react'

export interface AppNavigationItem {
  to: string
  labelKey: string
  icon: typeof Home
  end?: boolean
}

export interface AppNavigationSection {
  labelKey: string
  items: AppNavigationItem[]
}

const primaryItems: AppNavigationItem[] = [
  { to: '/', labelKey: 'navigation.overview', icon: Home, end: true },
  { to: '/events', labelKey: 'navigation.events', icon: Bell },
  { to: '/integrations', labelKey: 'navigation.integrations', icon: Plug, end: true },
  { to: '/notification-channels', labelKey: 'navigation.notificationChannels', icon: Activity, end: true },
]

const adminItems: AppNavigationItem[] = [
  { to: '/admin', labelKey: 'navigation.system', icon: ShieldCheck, end: true },
  { to: '/admin/logs', labelKey: 'navigation.apiLogs', icon: FileText, end: true },
  { to: '/admin/application-logs', labelKey: 'navigation.applicationLogs', icon: Activity, end: true },
]

export const profileItem: AppNavigationItem = {
  to: '/profile',
  labelKey: 'navigation.profile',
  icon: User,
  end: true,
}

export function getNavigationSections(isAdmin: boolean): AppNavigationSection[] {
  return [
    { labelKey: 'navigation.primary', items: primaryItems },
    ...(isAdmin ? [{ labelKey: 'navigation.administration', items: adminItems }] : []),
  ]
}

export function getCurrentPageTitleKey(pathname: string) {
  if (/^\/events\/[^/]+$/.test(pathname)) return 'navigation.eventDetail'
  if (pathname === '/events') return 'navigation.events'
  if (pathname === '/integrations') return 'navigation.integrations'
  if (pathname === '/notification-channels') return 'navigation.notificationChannels'
  if (pathname === '/profile') return 'navigation.profile'
  if (pathname === '/admin/logs') return 'navigation.apiLogs'
  if (pathname === '/admin/application-logs') return 'navigation.applicationLogs'
  if (pathname === '/admin') return 'navigation.system'
  if (pathname === '/forbidden') return 'navigation.forbidden'
  if (pathname === '/') return 'navigation.overview'
  return 'navigation.notFound'
}
