import React, { useState, useMemo } from 'react'
import * as Checkbox from '@radix-ui/react-checkbox'
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
  selectable?: boolean
  onSelectionChange?: (selectedItems: T[]) => void
  getItemId?: (item: T) => string
}

export function SearchableTable<T extends object>({
  data,
  columns,
  searchFields,
  searchPlaceholder = "Search...",
  emptyMessage = "No data found",
  className = "",
  selectable = false,
  onSelectionChange,
  getItemId = (item: T) => (item as any).id || JSON.stringify(item)
}: SearchableTableProps<T>) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortField, setSortField] = useState<keyof T | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set())

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

  const handleSelectAll = () => {
    const allIds = new Set(filteredAndSortedData.map(item => getItemId(item)))
    if (selectedItems.size === allIds.size && [...selectedItems].every(id => allIds.has(id))) {
      setSelectedItems(new Set())
    } else {
      setSelectedItems(allIds)
    }
  }

  const handleSelectItem = (item: T) => {
    const itemId = getItemId(item)
    const newSelected = new Set(selectedItems)
    if (newSelected.has(itemId)) {
      newSelected.delete(itemId)
    } else {
      newSelected.add(itemId)
    }
    setSelectedItems(newSelected)
  }

  const isSelected = (item: T) => selectedItems.has(getItemId(item))
  const isAllSelected = filteredAndSortedData.length > 0 && filteredAndSortedData.every(item => isSelected(item))
  const isIndeterminate = selectedItems.size > 0 && !isAllSelected

  // Update parent component when selection changes
  React.useEffect(() => {
    if (onSelectionChange) {
      const selectedData = filteredAndSortedData.filter(item => isSelected(item))
      onSelectionChange(selectedData)
    }
  }, [selectedItems, filteredAndSortedData, onSelectionChange])

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
          <div className="overflow-x-auto">
            <table className="w-full min-w-[800px]">
            <thead className="bg-secondary">
              <tr>
                {selectable && (
                  <th className="px-4 py-3 text-left text-sm font-medium text-foreground w-12">
                    <Checkbox.Root
                      checked={isAllSelected}
                      onCheckedChange={handleSelectAll}
                      className="flex h-4 w-4 items-center justify-center rounded border border-gray-300 bg-white data-[state=checked]:bg-primary data-[state=checked]:border-primary data-[state=indeterminate]:bg-primary data-[state=indeterminate]:border-primary"
                    >
                      <Checkbox.Indicator className="flex items-center justify-center text-white">
                        {isIndeterminate ? (
                          <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                          </svg>
                        ) : (
                          <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                        )}
                      </Checkbox.Indicator>
                    </Checkbox.Root>
                  </th>
                )}
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
                    colSpan={selectable ? columns.length + 1 : columns.length}
                    className="px-4 py-8 text-center text-muted-foreground"
                  >
                    {emptyMessage}
                  </td>
                </tr>
              ) : (
                filteredAndSortedData.map((item, index) => (
                  <tr key={index} className="hover:bg-muted">
                    {selectable && (
                      <td className="px-4 py-3 text-sm w-12">
                        <Checkbox.Root
                          checked={isSelected(item)}
                          onCheckedChange={() => handleSelectItem(item)}
                          className="flex h-4 w-4 items-center justify-center rounded border border-gray-300 bg-white data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                        >
                          <Checkbox.Indicator className="flex items-center justify-center text-white">
                            <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
                              <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                            </svg>
                          </Checkbox.Indicator>
                        </Checkbox.Root>
                      </td>
                    )}
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
          </div>
        </ScrollArea>
      </Card>

      {/* Results count */}
      <div className="text-sm text-muted-foreground">
        Showing {filteredAndSortedData.length} of {data.length} results
      </div>
    </div>
  )
}
