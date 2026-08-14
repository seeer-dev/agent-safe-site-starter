import { describe, it, expect } from 'vitest'
import {
  parseImageHeader,
  MIN_HEADER_BYTES,
  JPEG_SCAN_MAX_BYTES,
} from './image-header'
import type { ImageHeaderInfo } from './media-types'

// --- Helper: build synthetic image headers ---

/** Maximum data payload per JPEG segment (16-bit length field includes
 *  the 2 length bytes themselves, so max data = 65535 - 2 = 65533). */
const MAX_SEG_DATA = 65533

/** Build a minimal JPEG header with SOF marker at a given offset.
 *  Throws if any segment's data exceeds 65533 bytes (16-bit length limit). */
function buildJPEGHeader(
  width: number,
  height: number,
  appSegments: { marker: number; data: number[] }[] = [],
  totalSize = MIN_HEADER_BYTES,
  sofMarker: 0xc0 | 0xc1 | 0xc2 | 0xc3 = 0xc0,
): Uint8Array {
  // Validate segment data sizes — JPEG segment length is 16-bit
  for (const seg of appSegments) {
    if (seg.data.length > MAX_SEG_DATA) {
      throw new Error(
        `JPEG segment data length ${seg.data.length} exceeds 16-bit limit (${MAX_SEG_DATA} bytes max)`,
      )
    }
  }

  const buf = new Uint8Array(totalSize)
  // SOI
  buf[0] = 0xff
  buf[1] = 0xd8

  let i = 2
  for (const seg of appSegments) {
    buf[i] = 0xff
    buf[i + 1] = seg.marker
    const segLen = seg.data.length + 2 // length field includes itself
    buf[i + 2] = (segLen >> 8) & 0xff
    buf[i + 3] = segLen & 0xff
    for (let j = 0; j < seg.data.length; j++) {
      buf[i + 4 + j] = seg.data[j]
    }
    i += 2 + segLen
  }

  // SOF marker (SOF0=0xC0 baseline, SOF2=0xC2 progressive, etc.)
  buf[i] = 0xff
  buf[i + 1] = sofMarker
  // SOF segment: length(2) + precision(1) + height(2) + width(2) + components(1) + ...
  const sofLen = 17 // typical for 3-component JPEG
  buf[i + 2] = (sofLen >> 8) & 0xff
  buf[i + 3] = sofLen & 0xff
  buf[i + 4] = 8 // precision
  buf[i + 5] = (height >> 8) & 0xff
  buf[i + 6] = height & 0xff
  buf[i + 7] = (width >> 8) & 0xff
  buf[i + 8] = width & 0xff
  buf[i + 9] = 3 // components

  return buf
}

/** Build a minimal PNG header with IHDR chunk. */
function buildPNGHeader(width: number, height: number): Uint8Array {
  const buf = new Uint8Array(64)
  // PNG signature
  buf[0] = 0x89; buf[1] = 0x50; buf[2] = 0x4e; buf[3] = 0x47
  buf[4] = 0x0d; buf[5] = 0x0a; buf[6] = 0x1a; buf[7] = 0x0a
  // IHDR chunk length (4 bytes, big-endian) = 13
  buf[8] = 0; buf[9] = 0; buf[10] = 0; buf[11] = 13
  // Chunk type "IHDR"
  buf[12] = 0x49; buf[13] = 0x48; buf[14] = 0x44; buf[15] = 0x52
  // Width (4 bytes, big-endian)
  const w = new DataView(buf.buffer)
  w.setUint32(16, width, false)
  // Height (4 bytes, big-endian)
  w.setUint32(20, height, false)
  return buf
}

/** Build a minimal GIF header. */
function buildGIFHeader(width: number, height: number, version: '87a' | '89a' = '89a'): Uint8Array {
  const buf = new Uint8Array(32)
  // GIF signature
  buf[0] = 0x47; buf[1] = 0x49; buf[2] = 0x46; buf[3] = 0x38
  buf[4] = version === '87a' ? 0x37 : 0x39
  buf[5] = 0x61
  // Width (2 bytes, little-endian)
  buf[6] = width & 0xff; buf[7] = (width >> 8) & 0xff
  // Height (2 bytes, little-endian)
  buf[8] = height & 0xff; buf[9] = (height >> 8) & 0xff
  return buf
}

describe('parseImageHeader', () => {
  // --- JPEG ---

  it('parses simple JPEG with SOF0 at small offset', () => {
    const buf = buildJPEGHeader(1920, 1080)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(1920)
    expect(info.height).toBe(1080)
  })

  it('parses JPEG with large APP metadata before SOF (offset > 64 KiB)', () => {
    // Use multiple legal-size APP segments (each <= 65533 bytes) so
    // the total offset to SOF exceeds 64 KiB. This simulates real-world
    // JPEGs with large EXIF + ICC + XMP metadata.
    // 2 segments of 60 KiB each = 120 KiB offset (well past 64 KiB).
    const seg1Data = new Array(60 * 1024).fill(0x41) // APP1 EXIF
    const seg2Data = new Array(60 * 1024).fill(0x42) // APP2 ICC
    const buf = buildJPEGHeader(4000, 3000, [
      { marker: 0xe1, data: seg1Data },
      { marker: 0xe2, data: seg2Data },
    ], 200 * 1024)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(4000)
    expect(info.height).toBe(3000)
  })

  it('parses JPEG with multiple large APP segments totaling > 64 KiB offset', () => {
    // 3 segments: 50 KiB + 55 KiB + 60 KiB = 165 KiB offset
    const app1Data = new Array(50 * 1024).fill(0x42) // EXIF
    const app2Data = new Array(55 * 1024).fill(0x43) // ICC
    const app1XmpData = new Array(60 * 1024).fill(0x44) // XMP
    const buf = buildJPEGHeader(3840, 2160, [
      { marker: 0xe1, data: app1Data },
      { marker: 0xe2, data: app2Data },
      { marker: 0xe1, data: app1XmpData },
    ], 200 * 1024)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(3840)
    expect(info.height).toBe(2160)
  })

  it('parses progressive JPEG with SOF2 (0xC2) marker', () => {
    const buf = buildJPEGHeader(3840, 2160, [], MIN_HEADER_BYTES, 0xc2)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(3840)
    expect(info.height).toBe(2160)
  })

  it('parses progressive JPEG with SOF2 after large APP metadata', () => {
    const seg1Data = new Array(60 * 1024).fill(0x41)
    const seg2Data = new Array(60 * 1024).fill(0x42)
    const buf = buildJPEGHeader(2560, 1440, [
      { marker: 0xe1, data: seg1Data },
      { marker: 0xe2, data: seg2Data },
    ], 200 * 1024, 0xc2)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(2560)
    expect(info.height).toBe(1440)
  })

  it('returns sentinel when initial 1 KiB slice lacks SOF (large APP metadata)', () => {
    // Build a complete, legal JPEG with two 60 KiB APP segments so the
    // SOF marker sits at offset ~120 KiB (well past 64 KiB). Then pass
    // only the first 1024 bytes to the parser — SOF is not in this slice.
    const seg1Data = new Array(60 * 1024).fill(0x41)
    const seg2Data = new Array(60 * 1024).fill(0x42)
    const full = buildJPEGHeader(4000, 3000, [
      { marker: 0xe1, data: seg1Data },
      { marker: 0xe2, data: seg2Data },
    ], 200 * 1024)
    // Slice to simulate the initial MIN_HEADER_BYTES read
    const initialSlice = full.slice(0, MIN_HEADER_BYTES)
    const info = parseImageHeader(initialSlice)
    expect(info.format).toBe('jpeg')
    // SOF not in this slice — sentinel
    expect(info.width).toBe(0)
    expect(info.height).toBe(0)
  })

  it('returns sentinel when segment length extends past buffer (truncated segment)', () => {
    // Build a JPEG where the APP1 segment length field claims 5000 bytes
    // but the buffer is only 64 bytes — the segment body is truncated.
    // The parser should return a sentinel (cannot skip past the segment).
    const buf = new Uint8Array(64)
    buf[0] = 0xff; buf[1] = 0xd8 // SOI
    buf[2] = 0xff; buf[3] = 0xe1 // APP1 marker
    buf[4] = (5000 >> 8) & 0xff; buf[5] = 5000 & 0xff // length = 5000 (claims 5000 bytes)
    // Rest of buffer is zeros — segment body is truncated
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    // Cannot skip past the truncated segment — sentinel
    expect(info.width).toBe(0)
    expect(info.height).toBe(0)
  })

  it('throws on JPEG with invalid segment length (< 2)', () => {
    const buf = new Uint8Array(32)
    buf[0] = 0xff; buf[1] = 0xd8 // SOI
    buf[2] = 0xff; buf[3] = 0xe1 // APP1
    buf[4] = 0; buf[5] = 1 // segment length = 1 (invalid, min is 2)
    expect(() => parseImageHeader(buf)).toThrow(/segment length.*invalid/)
  })

  it('throws on JPEG with zero width in SOF', () => {
    const buf = buildJPEGHeader(0, 1080)
    expect(() => parseImageHeader(buf)).toThrow(/zero dimension/)
  })

  it('throws on JPEG with zero height in SOF', () => {
    const buf = buildJPEGHeader(1920, 0)
    expect(() => parseImageHeader(buf)).toThrow(/zero dimension/)
  })

  it('handles padding 0xFF bytes before markers', () => {
    const buf = new Uint8Array(64)
    buf[0] = 0xff; buf[1] = 0xd8 // SOI
    // Padding 0xFF bytes
    buf[2] = 0xff; buf[3] = 0xff; buf[4] = 0xff
    // APP1 marker after padding
    buf[5] = 0xe1
    buf[6] = 0; buf[7] = 4 // segment length = 4 (2 bytes length + 2 bytes data)
    buf[8] = 0x42; buf[9] = 0x43 // dummy data
    // SOF0
    buf[10] = 0xff; buf[11] = 0xc0
    buf[12] = 0; buf[13] = 17 // segment length
    buf[14] = 8 // precision
    buf[15] = 0x04; buf[16] = 0x38 // height = 1080
    buf[17] = 0x07; buf[18] = 0x80 // width = 1920
    buf[19] = 3 // components
    const info = parseImageHeader(buf)
    expect(info.format).toBe('jpeg')
    expect(info.width).toBe(1920)
    expect(info.height).toBe(1080)
  })

  // --- PNG ---

  it('parses PNG with valid IHDR', () => {
    const buf = buildPNGHeader(1920, 1080)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('png')
    expect(info.width).toBe(1920)
    expect(info.height).toBe(1080)
  })

  it('parses PNG with large dimensions (uses DataView, no signed overflow)', () => {
    // 3000x3000 = 9 MP, well within limits but tests DataView path
    const buf = buildPNGHeader(3000, 3000)
    const info = parseImageHeader(buf)
    expect(info.format).toBe('png')
    expect(info.width).toBe(3000)
    expect(info.height).toBe(3000)
  })

  it('throws on PNG with zero width', () => {
    const buf = buildPNGHeader(0, 100)
    expect(() => parseImageHeader(buf)).toThrow(/zero dimension/)
  })

  it('throws on PNG with zero height', () => {
    const buf = buildPNGHeader(100, 0)
    expect(() => parseImageHeader(buf)).toThrow(/zero dimension/)
  })

  it('throws on PNG with non-IHDR first chunk', () => {
    const buf = buildPNGHeader(100, 100)
    // Overwrite chunk type to "pHYs" instead of "IHDR"
    buf[12] = 0x70; buf[13] = 0x48; buf[14] = 0x59; buf[15] = 0x73
    expect(() => parseImageHeader(buf)).toThrow(/not IHDR/)
  })

  it('throws on PNG header too small (< 24 bytes)', () => {
    const buf = new Uint8Array(20)
    // PNG signature
    buf[0] = 0x89; buf[1] = 0x50; buf[2] = 0x4e; buf[3] = 0x47
    buf[4] = 0x0d; buf[5] = 0x0a; buf[6] = 0x1a; buf[7] = 0x0a
    expect(() => parseImageHeader(buf)).toThrow(/too small/)
  })

  // --- GIF ---

  it('parses GIF89a', () => {
    const buf = buildGIFHeader(800, 600, '89a')
    const info = parseImageHeader(buf)
    expect(info.format).toBe('gif')
    expect(info.width).toBe(800)
    expect(info.height).toBe(600)
  })

  it('parses GIF87a', () => {
    const buf = buildGIFHeader(400, 300, '87a')
    const info = parseImageHeader(buf)
    expect(info.format).toBe('gif')
    expect(info.width).toBe(400)
    expect(info.height).toBe(300)
  })

  it('throws on GIF with zero dimension', () => {
    const buf = buildGIFHeader(0, 100)
    expect(() => parseImageHeader(buf)).toThrow(/zero dimension/)
  })

  // --- Unknown format ---

  it('returns unknown for non-image data', () => {
    const buf = new Uint8Array([0x48, 0x54, 0x54, 0x50, 0x2f, 0x31, 0x2e, 0x31])
    const info = parseImageHeader(buf)
    expect(info.format).toBe('unknown')
  })

  it('returns unknown for WebP', () => {
    // WebP: RIFF....WEBP
    const buf = new Uint8Array(32)
    buf[0] = 0x52; buf[1] = 0x49; buf[2] = 0x46; buf[3] = 0x46 // RIFF
    buf[8] = 0x57; buf[9] = 0x45; buf[10] = 0x42; buf[11] = 0x50 // WEBP
    const info = parseImageHeader(buf)
    expect(info.format).toBe('unknown')
  })

  it('throws on header too small (< 8 bytes)', () => {
    const buf = new Uint8Array(4)
    expect(() => parseImageHeader(buf)).toThrow(/too small/)
  })

  // --- JPEG scan constants ---

  it('JPEG_SCAN_MAX_BYTES is 1 MiB', () => {
    expect(JPEG_SCAN_MAX_BYTES).toBe(1024 * 1024)
  })
})
