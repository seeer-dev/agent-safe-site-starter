import type { RoleKey, Role } from '@/lib/types';

// 角色 → capability 集合（用來示範 deniedMode:"hide"）
export const ROLES: Record<RoleKey, Role> = {
  owner: {
    key: 'owner',
    label: 'Owner',
    caps: [
      'twcommerce.read',
      'twcommerce.create',
      'twcommerce.update',
      'twcommerce.delete',
      'orders.returns',
      'inventory.adjust',
      'content.read',
      'content.create',
      'content.update',
      'content.approve',
      'content.publish',
      'staff.read',
      'staff.update',
    ],
  },
  manager: {
    key: 'manager',
    label: 'Manager',
    caps: [
      'twcommerce.read',
      'twcommerce.create',
      'twcommerce.update',
      'twcommerce.delete',
      'orders.returns',
      'inventory.adjust',
      'content.read',
      'content.create',
      'content.update',
      'content.approve',
      'content.publish',
    ],
  },
  readonly: {
    key: 'readonly',
    label: '唯讀',
    caps: ['twcommerce.read', 'content.read', 'staff.read'],
  },
  nocontent: {
    key: 'nocontent',
    label: '無內容權限',
    caps: [
      'twcommerce.read',
      'twcommerce.create',
      'twcommerce.update',
      'twcommerce.delete',
      'staff.read',
    ],
  },
};
