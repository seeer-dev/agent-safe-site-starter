import type { StateMachineFlow } from '@/lib/types';

// 訂單的兩條獨立狀態機（給頁面上的說明條用）
export const MACHINES: Record<string, StateMachineFlow[]> = {
  'minimal-cart-orders': [
    {
      t: '履約',
      flow: ['pending', 'processing', 'shipped', 'delivered'],
      alt: 'cancelled（pending / processing 時可取消）',
    },
    {
      t: '退貨',
      flow: ['requested', 'approved', 'received'],
      alt: 'rejected（待審時可駁回）',
    },
  ],
};
