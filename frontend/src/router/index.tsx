import { createBrowserRouter } from 'react-router-dom'
import { lazy } from 'react'
import { AppLayout } from '@/components/layout/app-layout'
import { PublicEventPage } from '@/pages/public-event-page'
import { GuestOnly, RequireAdmin, RequireAuth } from './guards'

const AdminPage = lazy(() => import('@/pages/admin-page').then((module) => ({ default: module.AdminPage })))
const HomePage = lazy(() => import('@/pages/home-page').then((module) => ({ default: module.HomePage })))
const LoginPage = lazy(() => import('@/pages/login-page').then((module) => ({ default: module.LoginPage })))
const ProfilePage = lazy(() => import('@/pages/profile-page').then((module) => ({ default: module.ProfilePage })))
const RegisterPage = lazy(() => import('@/pages/register-page').then((module) => ({ default: module.RegisterPage })))
const ForbiddenPage = lazy(() => import('@/pages/status-pages').then((module) => ({ default: module.ForbiddenPage })))
const NotFoundPage = lazy(() => import('@/pages/status-pages').then((module) => ({ default: module.NotFoundPage })))
const EventsPage = lazy(() => import('@/pages/events-page').then((module) => ({ default: module.EventsPage })))
const EventDetailPage = lazy(() => import('@/pages/event-detail-page').then((module) => ({ default: module.EventDetailPage })))
const IntegrationsPage = lazy(() => import('@/pages/integrations-page').then((module) => ({ default: module.IntegrationsPage })))
const NotificationChannelsPage = lazy(() => import('@/pages/notification-channels-page').then((module) => ({ default: module.NotificationChannelsPage })))
const AdminLogsPage = lazy(() => import('@/pages/admin-logs-page').then((module) => ({ default: module.AdminLogsPage })))
const AdminApplicationLogsPage = lazy(() => import('@/pages/admin-application-logs-page').then((module) => ({ default: module.AdminApplicationLogsPage })))
export const router = createBrowserRouter([
  { path: 'public/events/:token', element: <PublicEventPage /> },
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { element: <GuestOnly />, children: [{ path: 'login', element: <LoginPage /> }, { path: 'register', element: <RegisterPage /> }] },
      { element: <RequireAuth />, children: [{ path: 'profile', element: <ProfilePage /> }, { path: 'events', element: <EventsPage /> }, { path: 'events/:id', element: <EventDetailPage /> }, { path: 'integrations', element: <IntegrationsPage /> }, { path: 'notification-channels', element: <NotificationChannelsPage /> }, { path: 'forbidden', element: <ForbiddenPage /> }] },
      { element: <RequireAuth />, children: [{ element: <RequireAdmin />, children: [{ path: 'admin', element: <AdminPage /> }, { path: 'admin/logs', element: <AdminLogsPage /> }, { path: 'admin/application-logs', element: <AdminApplicationLogsPage /> }] }] },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
