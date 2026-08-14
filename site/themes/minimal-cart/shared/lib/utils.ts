import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatNTD(n: number): string {
  return `NT$${n.toLocaleString('zh-TW')}`
}

export function formatNumber(n: number): string {
  return n.toLocaleString('zh-TW')
}
