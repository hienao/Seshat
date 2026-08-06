import { createBrowserRouter } from 'react-router-dom'
import { lazy } from 'react'
import { AppLayout } from '@/components/layout/app-layout'
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
const AdminLogsPage = lazy(() => import('@/pages/admin-logs-page').then((module) => ({ default: module.AdminLogsPage })))

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { element: <GuestOnly />, children: [{ path: 'login', element: <LoginPage /> }, { path: 'register', element: <RegisterPage /> }] },
      { element: <RequireAuth />, children: [{ path: 'profile', element: <ProfilePage /> }, { path: 'events', element: <EventsPage /> }, { path: 'events/:id', element: <EventDetailPage /> }, { path: 'integrations', element: <IntegrationsPage /> }, { path: 'forbidden', element: <ForbiddenPage /> }] },
      { element: <RequireAuth />, children: [{ element: <RequireAdmin />, children: [{ path: 'admin', element: <AdminPage /> }, { path: 'admin/logs', element: <AdminLogsPage /> }] }] },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
