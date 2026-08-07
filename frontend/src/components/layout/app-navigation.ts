import { Bell, FileText, Home, Plug, ShieldCheck, User } from '@appica/icons-react'

export interface AppNavigationItem {
  to: string
  label: string
  icon: typeof Home
  end?: boolean
}

export interface AppNavigationSection {
  label: string
  items: AppNavigationItem[]
}

const primaryItems: AppNavigationItem[] = [
  { to: '/', label: '概览', icon: Home, end: true },
  { to: '/events', label: '消息流', icon: Bell },
  { to: '/integrations', label: '接入实例', icon: Plug, end: true },
]

const adminItems: AppNavigationItem[] = [
  { to: '/admin', label: '系统管理', icon: ShieldCheck, end: true },
  { to: '/admin/logs', label: '接口日志', icon: FileText, end: true },
]

export const profileItem: AppNavigationItem = {
  to: '/profile',
  label: '个人中心',
  icon: User,
  end: true,
}

export function getNavigationSections(isAdmin: boolean): AppNavigationSection[] {
  return [
    { label: '主要功能', items: primaryItems },
    ...(isAdmin ? [{ label: '系统管理', items: adminItems }] : []),
  ]
}

export function getCurrentPageTitle(pathname: string) {
  if (/^\/events\/[^/]+$/.test(pathname)) return '消息详情'
  if (pathname === '/events') return '消息流'
  if (pathname === '/integrations') return '接入实例'
  if (pathname === '/profile') return '个人中心'
  if (pathname === '/admin/logs') return '接口日志'
  if (pathname === '/admin') return '系统管理'
  if (pathname === '/forbidden') return '无权访问'
  if (pathname === '/') return '概览'
  return '页面未找到'
}
