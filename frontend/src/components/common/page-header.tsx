import type { ReactNode } from 'react'

export function PageHeader({ eyebrow, title, description, action }: {
  eyebrow?: string
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
      <div>
        {eyebrow && <p className="mb-2 text-xs font-bold uppercase tracking-[0.22em] text-emerald-700 dark:text-emerald-400">{eyebrow}</p>}
        <h1 className="text-3xl font-bold tracking-[-0.035em] text-neutral-950 dark:text-white sm:text-4xl">{title}</h1>
        <p className="mt-2 max-w-2xl text-neutral-600 dark:text-neutral-400">{description}</p>
      </div>
      {action}
    </div>
  )
}
