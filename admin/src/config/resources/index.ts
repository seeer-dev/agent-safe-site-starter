import type { ResourceDef } from '@/lib/types';
import { productsResource } from './products';
import { ordersResource } from './orders';
import { membersResource } from './members';
import { promosResource } from './promos';
import { contentResource } from './content';
import { paymentMethodsResource } from './payment-methods';
import { shippingMethodsResource } from './shipping-methods';
import { staffResource } from './staff';

export const RES: Record<string, ResourceDef> = {
  'minimal-cart-products': productsResource,
  'minimal-cart-orders': ordersResource,
  'minimal-cart-members': membersResource,
  'minimal-cart-promos': promosResource,
  'minimal-cart-content': contentResource,
  'tw-commerce.methods': paymentMethodsResource,
  'minimal-cart-shipping': shippingMethodsResource,
  staff: staffResource,
};
