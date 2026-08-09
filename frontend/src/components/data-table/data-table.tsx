import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from '@tanstack/react-table'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@appica/ui-react/table'
import { EmptyState } from '@/components/common/feedback'
import { useTranslation } from 'react-i18next'

export function DataTable<T>({ data, columns, emptyText }: {
  data: T[]
  columns: ColumnDef<T, unknown>[]
  emptyText?: string
}) {
  const { t } = useTranslation()
  const table = useReactTable({ data, columns, getCoreRowModel: getCoreRowModel() })
  if (!data.length) return <EmptyState title={emptyText ?? t('common.feedback.noData')} />

  return (
    <div className="overflow-x-auto">
      <Table hoverableRows>
        <TableHeader>
          {table.getHeaderGroups().map((group) => (
            <TableRow key={group.id}>
              {group.headers.map((header) => (
                <TableHead key={header.id}>{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}</TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow key={row.id}>
              {row.getVisibleCells().map((cell) => <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>)}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
