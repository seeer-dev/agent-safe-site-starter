import { describe, expect, it } from 'vitest'
import { orderKeys } from './query-keys'

describe('order query keys', () => {
  it('includes the effective site in the cache identity', () => {
    expect(orderKeys.detail('site-a', 'order-1')).not.toEqual(orderKeys.detail('site-b', 'order-1'))
  })
})
