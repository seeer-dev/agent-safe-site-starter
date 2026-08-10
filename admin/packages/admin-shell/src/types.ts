import type { Component } from 'vue'

export type ContentWidth = 'locked' | 'fluid'

export interface ShellNavItem {
  key: string
  label: string
  href?: string | undefined
  icon?: Component | undefined
  current?: boolean | undefined
  disabled?: boolean | undefined
  disabledTitle?: string | undefined
  disabledNote?: string | undefined
  badge?: string | undefined
  sub?: string | undefined
  children?: ShellNavItem[] | undefined
  branchOpen?: boolean | undefined
}

export interface ShellNavGroup {
  key: string
  label: string
  items: ShellNavItem[]
}

export interface ShellNavigationModel {
  groups: ShellNavGroup[]
  footer: ShellNavItem[]
  mobile: ShellNavItem[]
}

export interface ShellBrand {
  logo: string
  label: string
  subtitle: string
}

export interface AdminShellCopy {
  primaryNavigation: string
  expandSidebar: string
  collapseSidebar: string
  openSidebar: string
  search: string
  mobilePrimaryNavigation: string
  moreNavigation: string
  more: string
}
