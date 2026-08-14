import {
  Truck, Store, RotateCcw, ShieldCheck, Sparkles, Gift, Clock,
  Package, Layers, type LucideIcon,
} from 'lucide-vue-next'

const ICON_MAP: Record<string, LucideIcon> = {
  truck: Truck,
  store: Store,
  'rotate-ccw': RotateCcw,
  'shield-check': ShieldCheck,
  sparkles: Sparkles,
  gift: Gift,
  clock: Clock,
  package: Package,
  layers: Layers,
}

export function getIcon(name: string): LucideIcon {
  return ICON_MAP[name] ?? Sparkles
}
