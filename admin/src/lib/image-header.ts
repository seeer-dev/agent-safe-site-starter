// Pure TS image header parser. Reads only the minimal bytes needed to
// extract format and dimensions from JPEG, PNG, and GIF files. Never
// performs a full image decode — that is the server's job.
//
// All functions accept a Uint8Array of the file header (caller reads
// the first N bytes from the File/Blob). The parser scans for markers
// within the provided buffer and returns dimensions or throws if the
// header is malformed or the buffer is too small.

import type { ImageHeaderInfo } from './media-types'

/**
 * Minimum header bytes to read for the initial slice. JPEG files may
 * have large APP metadata segments (EXIF, ICC, XMP) before the SOF
 * marker, so the parser uses progressive slicing up to
 * JPEG_SCAN_MAX_BYTES if the SOF marker is not found in the initial
 * buffer.
 */
export const MIN_HEADER_BYTES = 1024

/**
 * Maximum bytes the JPEG parser will scan looking for the SOF marker.
 * Real-world JPEGs with large EXIF/ICC/XMP APP segments can push the
 * SOF marker past 64 KiB. We scan progressively up to 1 MiB, which
 * covers all practical metadata without reading the entire file.
 */
export const JPEG_SCAN_MAX_BYTES = 1024 * 1024

/**
 * Progressive slice growth step for JPEG scanning. When the SOF marker
 * is not found in the current buffer, the caller reads the next slice
 * of this size (doubling each time up to JPEG_SCAN_MAX_BYTES).
 */
export const JPEG_SLICE_STEP = 64 * 1024

/**
 * Parse image format and dimensions from a header buffer.
 * Supports JPEG (SOF0/SOF2 markers), PNG (IHDR chunk), and GIF (logical
 * screen descriptor). Returns format='unknown' for unrecognized magic.
 */
export function parseImageHeader(buf: Uint8Array): ImageHeaderInfo {
  if (buf.length < 8) {
    throw new Error('header too small (need at least 8 bytes)')
  }

  // JPEG: starts with FFD8FF
  if (buf[0] === 0xff && buf[1] === 0xd8 && buf[2] === 0xff) {
    return parseJPEGDimensions(buf)
  }

  // PNG: starts with 89504E470D0A1A0A
  if (
    buf[0] === 0x89 && buf[1] === 0x50 && buf[2] === 0x4e &&
    buf[3] === 0x47 && buf[4] === 0x0d && buf[5] === 0x0a &&
    buf[6] === 0x1a && buf[7] === 0x0a
  ) {
    return parsePNGDimensions(buf)
  }

  // GIF: starts with "GIF87a" or "GIF89a"
  if (
    buf[0] === 0x47 && buf[1] === 0x49 && buf[2] === 0x46 &&
    buf[3] === 0x38 && (buf[4] === 0x37 || buf[4] === 0x39) && buf[5] === 0x61
  ) {
    return parseGIFDimensions(buf)
  }

  return { format: 'unknown', width: 0, height: 0 }
}

/**
 * Parse JPEG dimensions by scanning for SOF0 (0xC0) through SOF3 (0xC3)
 * markers. Scans within the provided buffer only — does NOT decode.
 *
 * The scan handles:
 * - Padding 0xFF bytes between markers
 * - APPn and other variable-length segments (reads 2-byte length, skips)
 * - Standalone markers (RSTn, TEM, SOI) with no segment length
 * - Strict bounds checking on every read
 *
 * If the SOF marker is not found within the buffer, returns a sentinel
 * with width=0, height=0 so the caller knows to read more bytes
 * (progressive slicing). The caller checks for this case and reads
 * a larger slice up to JPEG_SCAN_MAX_BYTES.
 */
function parseJPEGDimensions(buf: Uint8Array): ImageHeaderInfo {
  let i = 2 // skip FFD8 (SOI marker)
  const len = buf.length

  while (i < len - 1) {
    // Every marker starts with 0xFF
    if (buf[i] !== 0xff) {
      i++
      continue
    }

    // Skip padding 0xFF bytes (multiple 0xFF before a marker is valid)
    while (i < len - 1 && buf[i] === 0xff) {
      i++
    }
    if (i >= len) break

    const marker = buf[i]
    i++ // move past the marker byte

    // SOF0 (0xC0) through SOF3 (0xC3) — carry frame dimensions
    if (marker >= 0xc0 && marker <= 0xc3) {
      // SOF segment: length(2) + precision(1) + height(2) + width(2)
      // We need at least 5 bytes after the marker byte for the
      // length + precision + height + width fields.
      if (i + 6 > len) {
        throw new Error('JPEG SOF segment truncated in header buffer')
      }
      // Read segment length (2 bytes, big-endian) — includes itself
      const segLen = (buf[i] << 8) | buf[i + 1]
      if (segLen < 8) {
        throw new Error(`JPEG SOF segment length ${segLen} too small (minimum 8 for precision+height+width)`)
      }
      // Height and width are at offset +3 and +5 from segment start
      const height = (buf[i + 3] << 8) | buf[i + 4]
      const width = (buf[i + 5] << 8) | buf[i + 6]
      if (width === 0 || height === 0) {
        throw new Error(`JPEG SOF has zero dimension (${width}x${height})`)
      }
      return { format: 'jpeg', width, height }
    }

    // Standalone markers with no segment length (RST0-7, TEM, SOI, EOI)
    if (
      (marker >= 0xd0 && marker <= 0xd7) || // RST0-RST7
      marker === 0x01 ||                     // TEM
      marker === 0xd8 ||                     // SOI (shouldn't appear again)
      marker === 0xd9                        // EOI
    ) {
      continue
    }

    // SOS (0xDA) — scan data starts here, SOF should have been found
    // before SOS. If we hit SOS without finding SOF, the file is
    // malformed or the buffer is too small.
    if (marker === 0xda) {
      // SOF not found before SOS — need more bytes or file is corrupt
      return { format: 'jpeg', width: 0, height: 0 }
    }

    // All other markers (APPn, DQT, DHT, COM, etc.) have a 2-byte
    // segment length. Read it and skip.
    if (i + 2 > len) {
      // Segment length would extend past buffer — need more bytes
      return { format: 'jpeg', width: 0, height: 0 }
    }
    const segLen = (buf[i] << 8) | buf[i + 1]
    // Segment length includes the 2 length bytes themselves
    if (segLen < 2) {
      throw new Error(`JPEG segment length ${segLen} invalid (minimum 2)`)
    }
    i += segLen
  }

  // SOF marker not found in the current buffer. Return sentinel so
  // the caller can read a larger slice (progressive slicing).
  return { format: 'jpeg', width: 0, height: 0 }
}

/**
 * Parse PNG dimensions from the IHDR chunk using DataView for safe
 * big-endian uint32 reads. Validates that width and height are > 0.
 */
function parsePNGDimensions(buf: Uint8Array): ImageHeaderInfo {
  // PNG: 8 sig + 4 length + 4 type("IHDR") + 4 width + 4 height
  if (buf.length < 24) {
    throw new Error('PNG header too small (need at least 24 bytes for IHDR)')
  }
  // Verify chunk type is IHDR (bytes 12-15)
  if (buf[12] !== 0x49 || buf[13] !== 0x48 || buf[14] !== 0x44 || buf[15] !== 0x52) {
    throw new Error('PNG first chunk is not IHDR')
  }
  // Use DataView for safe big-endian uint32 reads (avoids signed
  // bitwise OR issues with values >= 2^31)
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength)
  const width = dv.getUint32(16, false) // big-endian
  const height = dv.getUint32(20, false) // big-endian
  if (width === 0 || height === 0) {
    throw new Error(`PNG IHDR has zero dimension (${width}x${height})`)
  }
  return { format: 'png', width, height }
}

/** Parse GIF dimensions from the logical screen descriptor (bytes 6-9). */
function parseGIFDimensions(buf: Uint8Array): ImageHeaderInfo {
  // GIF: 6 sig + 2 width + 2 height
  if (buf.length < 10) {
    throw new Error('GIF header too small (need at least 10 bytes)')
  }
  const width = (buf[7] << 8) | buf[6]
  const height = (buf[9] << 8) | buf[8]
  if (width === 0 || height === 0) {
    throw new Error(`GIF has zero dimension (${width}x${height})`)
  }
  return { format: 'gif', width, height }
}

/**
 * Read the first MIN_HEADER_BYTES from a File/Blob as a Uint8Array.
 * For JPEG files where the SOF marker is not in the initial slice,
 * use readJPEGHeaderWithProgressiveSlicing instead.
 */
export async function readHeaderBytes(file: Blob): Promise<Uint8Array> {
  const slice = file.slice(0, MIN_HEADER_BYTES)
  const ab = await slice.arrayBuffer()
  return new Uint8Array(ab)
}

/**
 * Read a JPEG header with progressive slicing. Starts with
 * MIN_HEADER_BYTES and grows up to JPEG_SCAN_MAX_BYTES until the
 * SOF marker is found. Returns the parsed header info.
 *
 * This handles JPEGs with large APP metadata (EXIF, ICC, XMP) that
 * push the SOF marker past the initial 1 KiB slice.
 */
export async function readJPEGHeaderWithProgressiveSlicing(file: Blob): Promise<ImageHeaderInfo> {
  let readSize = MIN_HEADER_BYTES
  while (readSize <= JPEG_SCAN_MAX_BYTES) {
    const slice = file.slice(0, readSize)
    const ab = await slice.arrayBuffer()
    const buf = new Uint8Array(ab)
    const info = parseJPEGDimensions(buf)
    if (info.width > 0 && info.height > 0) {
      return info
    }
    // SOF not found — grow the slice
    if (readSize >= JPEG_SCAN_MAX_BYTES) break
    readSize = Math.min(readSize * 2, JPEG_SCAN_MAX_BYTES)
  }
  throw new Error(
    `JPEG SOF marker not found within ${JPEG_SCAN_MAX_BYTES / 1024} KiB (file may be corrupt or have excessive metadata)`,
  )
}

/**
 * Parse an image header from a File/Blob, handling JPEG progressive
 * slicing automatically. For non-JPEG formats, reads only
 * MIN_HEADER_BYTES.
 */
export async function parseImageHeaderFromFile(file: Blob): Promise<ImageHeaderInfo> {
  // Read initial header to detect format
  const buf = await readHeaderBytes(file)
  if (buf.length < 8) {
    throw new Error('file too small to contain a valid image header')
  }

  // Check magic bytes to determine format
  const isJPEG = buf[0] === 0xff && buf[1] === 0xd8 && buf[2] === 0xff
  if (isJPEG) {
    // Try initial parse — if SOF not found, use progressive slicing
    const info = parseJPEGDimensions(buf)
    if (info.width > 0 && info.height > 0) {
      return info
    }
    return readJPEGHeaderWithProgressiveSlicing(file)
  }

  // Non-JPEG: the initial buffer is sufficient (PNG IHDR at byte 16,
  // GIF dimensions at byte 6)
  return parseImageHeader(buf)
}
