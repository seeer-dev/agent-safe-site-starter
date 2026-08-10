/**
 * Content management domain model — CMS page list viewmodel.
 */

export type ContentStatus = 'published' | 'draft' | 'archived'

export interface ContentRow {
  id: string
  title: string
  placement: string
  locale: string
  updatedAt: string
  status: ContentStatus
}

export interface ContentPagination {
  page: number
  perPage: number
  total: number
  totalPages: number
}

export interface ContentListSummary {
  items: ContentRow[]
  pagination: ContentPagination
}
