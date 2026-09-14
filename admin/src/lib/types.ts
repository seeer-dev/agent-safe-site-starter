// TypeScript types for the config-driven admin system.
// Direct port of the mockup's PROFILE / RES / ROLES / TONE / LABEL / MACHINES
// constants in dashboard.html.

export type Capability = string;

export type RoleKey = 'owner' | 'manager' | 'readonly' | 'nocontent';

export interface Role {
  key: RoleKey;
  label: string;
  caps: Capability[];
}

export type SectionKey = 'primary' | 'secondary' | 'settings';

export interface NavChild {
  key: string;
  label: string;
  icon?: string;
}

export interface RouteDef {
  key: string;
  label: string;
  icon: string;
  section: SectionKey;
  component: 'MinimalCartDashboardPage' | 'ResourceListPage' | 'StoreSettingsPage';
  caps: Capability[];
  /** Optional sub-menu children. When present the nav item renders as a
   *  collapsible parent with inline children (expanded) and a hover flyout
   *  (collapsed). Currently unused — structure is reserved for future
   *  grouped navigation. */
  children?: NavChild[];
}

export type ColRender = 'mono' | 'badge' | 'number' | 'datetime';

export interface Col {
  k: string;
  l: string;
  r?: ColRender;
  /** Resolve raw cell values to display labels from an admin list endpoint
   *  (e.g. category slug -> category name). */
  optsSource?: { api: string; listKey?: string; value: string; label: string };
}

export type FilterWidget = 'select' | 'text';

export interface FilterDef {
  k: string;
  l: string;
  w: FilterWidget;
  opts?: [string, string][];
  /** For select filters: load options from an admin list endpoint. */
  optsSource?: { api: string; listKey?: string; value: string; label: string };
}

export type FieldWidget =
  | 'text'
  | 'textarea'
  | 'select'
  | 'number'
  | 'switch'
  | 'image'
  | 'tags'
  | 'datetime'
  | 'media-uploader'
  /** Editable sub-table of product variants: rows of
   *  {name, sku, price_delta, stock, sort_order}. */
  | 'variants';

export interface FieldDef {
  k: string;
  l: string;
  w: FieldWidget;
  req?: boolean;
  ro?: boolean;
  /** Read-only when editing an existing row; still editable on create. */
  roOnEdit?: boolean;
  /** When set on a number field, a blank value is sent as JSON null instead of 0. */
  nullable?: boolean;
  span?: number;
  opts?: (string | [string, string])[];
  /** For select fields: load options from an admin list endpoint.
   *  `api` returns `{<listKey>: [...]}` or a bare array; each entry's
   *  `value`/`label` keys pick the option value. Takes precedence over `opts`. */
  optsSource?: { api: string; listKey?: string; value: string; label: string };
  help?: string;
}

export interface FormSection {
  t: string;
  fields: FieldDef[];
}

export interface FormDef {
  title: string;
  readOnly?: boolean;
  sections: FormSection[];
}

export type RowActionVariant = 'pri' | 'sec' | 'danger' | 'ghost';

export interface RowAction {
  k: string;
  l: string;
  cap?: Capability;
  /** All-of capability gate: the action is visible/enabled only if the
   *  current principal holds EVERY capability in this list. Use this
   *  when the server requires multiple capabilities (e.g. restock
   *  requires both orders.returns AND inventory.adjust). When both cap
   *  and allCaps are set, they are merged into one all-of list (with
   *  de-duplication) — allCaps never shadows cap. */
  allCaps?: Capability[];
  variant?: RowActionVariant;
  form?: boolean;
  op?: string;
  payload?: Record<string, any>;
  showWhen?: string;
  confirm?: string;
  reason?: boolean;
  /** When set, the confirm-dialog reason input is sent under this JSON
   *  key instead of `note` (e.g. comment moderation sends `reply`). */
  reasonField?: string;
  /** Show the reason textarea without requiring a value. */
  reasonOptional?: boolean;
  /** Custom label for the reason textarea (default 原因). */
  reasonLabel?: string;
  expect?: string;
  /** If true, the confirm dialog shows a required datetime-local input
   *  whose value is converted to expiry_unix and sent with the request. */
  expiryInput?: boolean;
  /** If true, open the per-item restock modal instead of the confirm dialog.
   *  The restock modal shows each order line item with editable
   *  returned_quantity and restocked_quantity inputs (defaults 0, not full
   *  quantity) and a required reason. A stable idempotency key is generated
   *  once when the modal opens and reused across retries. */
  restockItems?: boolean;
}

export interface BulkAction {
  k: string;
  l: string;
  op: string;
  cap: Capability;
  payload?: Record<string, any>;
  confirm?: string;
  reason?: boolean;
  variant?: 'danger';
}

export interface OpsDef {
  list?: string;
  create?: string;
  update?: string;
  del?: string;
  get?: string;
  status?: string;
  bulk?: string;
  returnStatus?: string;
  publish?: string;
  approve?: string;
  restock?: string;
}

export interface ResourceApiDef {
  list: string; // GET path, e.g., '/admin/products'
  get?: string; // GET path with {id}, e.g., '/admin/products/{id}'
  create?: string; // POST path
  update?: string; // PUT path with {id}
  delete?: string; // DELETE path with {id}
  status?: string; // PATCH path with {id} for status updates
  returnStatus?: string; // PATCH path with {id} for return status
  publish?: string; // POST path with {id} for publish action
  approve?: string; // POST path with {id} for approve action
  restock?: string; // POST path with {id} for per-item restock action
}

export interface ResourceDef {
  label: string;
  desc: string;
  pageSize: number;
  ops: OpsDef;
  api?: ResourceApiDef;
  /** Transform a raw API row into the display/fixture shape used by the table. */
  rowMap?: (raw: Record<string, any>) => Record<string, any>;
  createCap?: Capability;
  updateCap?: Capability;
  /** Row field injected as expected_version on update (optimistic concurrency). */
  expectedVersionField?: string;
  cols: Col[];
  rowActions: RowAction[];
  bulkActions?: BulkAction[];
  filters: FilterDef[];
  form: FormDef;
  rows: Record<string, any>[];
}

export interface StateMachineFlow {
  t: string;
  flow: string[];
  alt: string;
}

export type ToneKey = 'neutral' | 'success' | 'warn' | 'danger' | 'info';
