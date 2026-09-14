// 格式化工具 — 與 reference src/lib/format.ts 相同行為

export function formatNTD(amount: number, withSymbol = true): string {
  const n = Math.round(amount).toLocaleString('zh-TW')
  return withSymbol ? `NT$${n}` : n
}

export function formatDate(date: string | number | Date | null | undefined): string {
  if (date == null || date === '') return '—'
  const d = date instanceof Date ? date : new Date(typeof date === 'number' ? date * 1000 : date)
  return new Intl.DateTimeFormat('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(d)
}

export function formatDateTime(date: string | number | Date | null | undefined): string {
  if (date == null || date === '') return '—'
  const d = date instanceof Date ? date : new Date(typeof date === 'number' ? date * 1000 : date)
  return new Intl.DateTimeFormat('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(d)
}

export function relativeTime(date: string | number | Date): string {
  const d = date instanceof Date ? date : new Date(typeof date === 'number' ? date * 1000 : date)
  const diff = Date.now() - d.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '剛剛'
  if (minutes < 60) return `${minutes} 分鐘前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小時前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return formatDate(d)
}

/** 是否在特價（original_price 為劃線原價） */
export function onSale(price: number, originalPrice?: number | null): boolean {
  return originalPrice != null && originalPrice > price
}
