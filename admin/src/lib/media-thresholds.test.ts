import { describe, it, expect } from 'vitest'
import {
  evaluateThresholds,
  validateCompressedOutput,
  SMALL_FILE_BYTES,
  COMPRESS_MAX_EDGE,
  MAX_FILE_BYTES,
  GIF_MAX_BYTES,
  GIF_MAX_DIM,
} from './media-thresholds'
import type { ImageHeaderInfo } from './media-types'

describe('evaluateThresholds', () => {
  // --- JPEG/PNG direct vs compress boundary ---

  it('direct: 1 MiB JPEG 2000px -> direct', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 2000, height: 1500 }
    const result = evaluateThresholds(1 * 1024 * 1024, header)
    expect(result.action).toBe('direct')
  })

  it('compress: 3 MiB JPEG 2000px -> compress (file > 2 MiB)', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 2000, height: 1500 }
    const result = evaluateThresholds(3 * 1024 * 1024, header)
    expect(result.action).toBe('compress')
  })

  it('compress: 1 MiB JPEG 3000px -> compress (longEdge > 2560)', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 3000, height: 2000 }
    const result = evaluateThresholds(1 * 1024 * 1024, header)
    expect(result.action).toBe('compress')
  })

  it('compress: 1 MiB PNG 3000px -> compress (longEdge > 2560)', () => {
    const header: ImageHeaderInfo = { format: 'png', width: 2000, height: 3000 }
    const result = evaluateThresholds(1 * 1024 * 1024, header)
    expect(result.action).toBe('compress')
  })

  it('direct: exactly 2 MiB JPEG 2560px -> direct (boundary)', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 2560, height: 1920 }
    const result = evaluateThresholds(SMALL_FILE_BYTES, header)
    expect(result.action).toBe('direct')
  })

  it('compress: 2 MiB + 1 byte JPEG 2560px -> compress', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 2560, height: 1920 }
    const result = evaluateThresholds(SMALL_FILE_BYTES + 1, header)
    expect(result.action).toBe('compress')
  })

  it('compress: 2 MiB JPEG 2561px -> compress', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 2561, height: 1920 }
    const result = evaluateThresholds(SMALL_FILE_BYTES, header)
    expect(result.action).toBe('compress')
  })

  it('direct: 500 KiB PNG 2560px -> direct', () => {
    const header: ImageHeaderInfo = { format: 'png', width: 2560, height: 2560 }
    const result = evaluateThresholds(500 * 1024, header)
    expect(result.action).toBe('direct')
  })

  // --- GIF pass/reject ---

  it('GIF direct: 5 MiB GIF 800x600 -> direct', () => {
    const header: ImageHeaderInfo = { format: 'gif', width: 800, height: 600 }
    const result = evaluateThresholds(5 * 1024 * 1024, header)
    expect(result.action).toBe('direct')
  })

  it('GIF reject: 11 MiB GIF 800x600 -> reject (size > 10 MiB)', () => {
    const header: ImageHeaderInfo = { format: 'gif', width: 800, height: 600 }
    const result = evaluateThresholds(11 * 1024 * 1024, header)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('GIF')
  })

  it('GIF reject: 5 MiB GIF 5000x5000 -> reject (dim > 4096)', () => {
    const header: ImageHeaderInfo = { format: 'gif', width: 5000, height: 5000 }
    const result = evaluateThresholds(5 * 1024 * 1024, header)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('GIF')
  })

  it('GIF direct: exactly 10 MiB GIF 4096x4096 -> direct (boundary)', () => {
    const header: ImageHeaderInfo = { format: 'gif', width: GIF_MAX_DIM, height: GIF_MAX_DIM }
    const result = evaluateThresholds(GIF_MAX_BYTES, header)
    expect(result.action).toBe('direct')
  })

  // --- Hard limits ---

  it('reject: 51 MiB JPEG -> reject (file > 50 MiB)', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 1000, height: 1000 }
    const result = evaluateThresholds(MAX_FILE_BYTES + 1, header)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('50 MiB')
  })

  it('reject: 9000x9000 JPEG -> reject (pixels > 64 MP)', () => {
    const header: ImageHeaderInfo = { format: 'jpeg', width: 9000, height: 9000 }
    const result = evaluateThresholds(1 * 1024 * 1024, header)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('64 MP')
  })

  it('reject: unknown format -> reject', () => {
    const header: ImageHeaderInfo = { format: 'unknown', width: 100, height: 100 }
    const result = evaluateThresholds(1000, header)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('unsupported')
  })

  // --- Compressed output validation ---

  it('validateCompressedOutput: 8 MiB 2000x2000 -> direct', () => {
    const result = validateCompressedOutput(8 * 1024 * 1024, 2000, 2000)
    expect(result.action).toBe('direct')
  })

  it('validateCompressedOutput: 11 MiB -> reject', () => {
    const result = validateCompressedOutput(11 * 1024 * 1024, 2000, 2000)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('10 MiB')
  })

  it('validateCompressedOutput: 5000px -> reject', () => {
    const result = validateCompressedOutput(5 * 1024 * 1024, 5000, 3000)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('4096')
  })

  it('validateCompressedOutput: width=0 -> reject', () => {
    const result = validateCompressedOutput(1000, 0, 100)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('invalid')
  })

  it('validateCompressedOutput: height=0 -> reject', () => {
    const result = validateCompressedOutput(1000, 100, 0)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('invalid')
  })

  it('validateCompressedOutput: negative dimensions -> reject', () => {
    const result = validateCompressedOutput(1000, -1, -1)
    expect(result.action).toBe('reject')
    if (result.action === 'reject') expect(result.reason).toContain('invalid')
  })
})
