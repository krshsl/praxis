import React, { useState, useMemo } from 'react'
import { Card } from 'components/ui/Card'
import { Button } from 'components/ui/Button'
import { ScrollArea } from 'components/ui/ScrollArea'
import { Input } from 'components/ui/Input'

export interface Column<T extends object> {
  key: keyof T
  label: string
  render?: (value: unknown, item: T) => React.ReactNode
  sortable?: boolean
}

export interface SearchableTableProps<T extends object> {
  data: T[]
  columns: Column<T>[]
  searchFields: (keyof T)[]
  searchPlaceholder?: string
  emptyMessage?: string
  className?: string
}

export function SearchableTable<T extends object>({
  data,
  columns,
  searchFields,
  searchPlaceholder = "Search...",
  emptyMessage = "No data found",
  className = ""
}: SearchableTableProps<T>) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortField, setSortField] = useState<keyof T | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')

  const filteredAndSortedData = useMemo(() => {
    let filtered = data

    // Filter by search term
    if (searchTerm) {
      filtered = data.filter(item =>
        searchFields.some(field => {
          const value = item[field]
          return value && value.toString().toLowerCase().includes(searchTerm.toLowerCase())
        })
      )
    }

    // Sort data
    if (sortField) {
      filtered = [...filtered].sort((a, b) => {
        const aValue = a[sortField]
        const bValue = b[sortField]
        
        if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1
        if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1
        return 0
      })
    }

    return filtered
  }, [data, searchTerm, sortField, sortDirection, searchFields])

  const handleSort = (field: keyof T) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('asc')
    }
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Search Bar */}
      <div className="flex items-center space-x-4">
        <div className="flex-1">
          <Input
            placeholder={searchPlaceholder}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        {searchTerm && (
          <Button
            onClick={() => setSearchTerm('')}
            variant="outline"
            size="sm"
          >
            Clear
          </Button>
        )}
      </div>

      {/* Table */}
      <Card className="overflow-hidden">
        <ScrollArea className="h-[400px]">
          <table className="w-full">
            <thead className="bg-secondary">
              <tr>
                {columns.map((column) => (
                  <th
                    key={String(column.key)}
                    className={`px-4 py-3 text-left text-sm font-medium text-foreground ${
                      column.sortable ? 'cursor-pointer hover:bg-muted' : ''
                    }`}
                    onClick={() => column.sortable && handleSort(column.key)}
                  >
                    <div className="flex items-center space-x-1">
                      <span>{column.label}</span>
                      {column.sortable && sortField === column.key && (
                        <span className="text-xs">
                          {sortDirection === 'asc' ? '↑' : '↓'}
                        </span>
                      )}
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filteredAndSortedData.length === 0 ? (
                <tr>
                  <td
                    colSpan={columns.length}
                    className="px-4 py-8 text-center text-muted-foreground"
                  >
                    {emptyMessage}
                  </td>
                </tr>
              ) : (
                filteredAndSortedData.map((item, index) => (
                  <tr key={index} className="hover:bg-muted">
                    {columns.map((column) => (
                      <td key={String(column.key)} className="px-4 py-3 text-sm">
                        {column.render
                          ? column.render(item[column.key], item)
                          : String(item[column.key] || '')}
                      </td>
                    ))}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </ScrollArea>
      </Card>

      {/* Results count */}
      <div className="text-sm text-muted-foreground">
        Showing {filteredAndSortedData.length} of {data.length} results
      </div>
    </div>
  )
}
