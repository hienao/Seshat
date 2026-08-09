import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { AppSidebar } from './app-sidebar'

vi.mock('@/features/auth/use-auth', () => ({
  useAuth: () => ({
    user: { username: 'admin' },
    isAdmin: true,
    logout: { mutate: vi.fn(), isPending: false },
  }),
}))

vi.mock('@/components/updates/app-version-status', () => ({
  AppVersionStatus: () => <span>v0.7.0 · Beta</span>,
}))

afterEach(cleanup)

describe('AppSidebar', () => {
  it('links the brand header to the Seshat GitHub project', () => {
    render(<MemoryRouter><AppSidebar className="flex" /></MemoryRouter>)

    const link = screen.getByRole('link', { name: '在 GitHub 打开 Seshat 项目' })
    expect(link).toHaveAttribute('href', 'https://github.com/hienao/Seshat')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })
})
