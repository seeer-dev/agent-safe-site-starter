// Threshold and decision logic for the media upload pipeline.
// All constants and decision functions are pure TS for testability.

import type { ImageHeaderInfo, ThresholdDecision } from './media-types'

/** Maximum file size: 50 MiB. */
export const MAX_FILE_BYTES = 50 * 1024 * 1024

/** Maximum pixel dimensions: 64 megapixels. */
export const MAX_PIXELS = 64_000_000

/** Direct upload size limit: 10 MiB (server verify limit). */
export const DIRECT_MAX_BYTES = 10 * 1024 * 1024

/** Direct upload dimension limit: 4096px per side. */
export const DIRECT_MAX_DIM = 4096

/** Small-file threshold for JPEG/PNG direct upload: 2 MiB. */
export const SMALL_FILE_BYTES = 2 * 1024 * 1024

/** Max edge for compressed output: 2560px. */
export const COMPRESS_MAX_EDGE = 2560

/** JPEG compression quality. */
export const JPEG_QUALITY = 0.82

/** GIF direct upload limit: 10 MiB (same as server verify limit). */
export const GIF_MAX_BYTES = 10 * 1024 * 1024

/** GIF direct upload dimension limit: 4096px per side. */
export const GIF_MAX_DIM = 4096

/**
 * Evaluate upload thresholds and decide whether to direct-upload,
 * compress, or reject. This is the single decision point for the
 * pipeline — the component and Worker both defer to this logic.
 *
 * Rules:
 * - Reject if format is unknown (not JPEG/PNG/GIF).
 * - Reject if file > 50 MiB.
 * - Reject if pixels (width*height) > 64 MP.
 * - GIF: never compress. Direct upload if <= 10 MiB AND <= 4096px.
 *   Otherwise reject.
 * - JPEG/PNG: direct upload ONLY if fileSize <= 2 MiB AND long edge
 *   <= 2560 (and within hard limits). If fileSize > 2 MiB OR
 *   longEdge > 2560, action is 'compress'. The caller checks Worker
 *   support and safely rejects if the Worker is unavailable.
 */
export function evaluateThresholds(
  fileSize: number,
  header: ImageHeaderInfo,
): ThresholdDecision {
  // Format check
  if (header.format === 'unknown') {
    return {
      action: 'reject',
      reason: 'unsupported format — only JPEG, PNG, and GIF are accepted',
    }
  }

  // File size hard limit
  if (fileSize > MAX_FILE_BYTES) {
    return {
      action: 'reject',
      reason: `file size ${(fileSize / 1024 / 1024).toFixed(1)} MiB exceeds 50 MiB limit`,
    }
  }

  // Pixel count hard limit
  const pixels = header.width * header.height
  if (pixels > MAX_PIXELS) {
    return {
      action: 'reject',
      reason: `image dimensions ${header.width}x${header.height} (${(pixels / 1_000_000).toFixed(1)} MP) exceed 64 MP limit`,
    }
  }

  if (header.format === 'gif') {
    // GIF: never Canvas/compress. Direct upload only if within limits.
    if (fileSize > GIF_MAX_BYTES) {
      return {
        action: 'reject',
        reason: `GIF size ${(fileSize / 1024 / 1024).toFixed(1)} MiB exceeds 10 MiB limit (GIF cannot be compressed)`,
      }
    }
    if (header.width > GIF_MAX_DIM || header.height > GIF_MAX_DIM) {
      return {
        action: 'reject',
        reason: `GIF dimensions ${header.width}x${header.height} exceed ${GIF_MAX_DIM}px limit (GIF cannot be compressed)`,
      }
    }
    return { action: 'direct' }
  }

  // JPEG or PNG
  const longEdge = Math.max(header.width, header.height)

  // Direct upload ONLY if both conditions hold:
  //   fileSize <= 2 MiB AND longEdge <= 2560
  // (and implicitly within hard limits since 2 MiB < 10 MiB and
  //  2560 < 4096, so the hard-limit checks above already passed).
  if (fileSize <= SMALL_FILE_BYTES && longEdge <= COMPRESS_MAX_EDGE) {
    return { action: 'direct' }
  }

  // Anything else needs compression. The caller checks Worker support
  // and safely rejects if unavailable — no main-thread fallback.
  return { action: 'compress' }
}

/**
 * Check if the compressed output is within server verify limits.
 * Called after the Worker returns a compressed blob.
 */
export function validateCompressedOutput(
  blobSize: number,
  width: number,
  height: number,
): ThresholdDecision {
  if (width <= 0 || height <= 0) {
    return {
      action: 'reject',
      reason: `compressed dimensions ${width}x${height} are invalid (must be > 0)`,
    }
  }
  if (blobSize > DIRECT_MAX_BYTES) {
    return {
      action: 'reject',
      reason: `compressed size ${(blobSize / 1024 / 1024).toFixed(1)} MiB still exceeds 10 MiB limit`,
    }
  }
  if (width > DIRECT_MAX_DIM || height > DIRECT_MAX_DIM) {
    return {
      action: 'reject',
      reason: `compressed dimensions ${width}x${height} exceed ${DIRECT_MAX_DIM}px limit`,
    }
  }
  return { action: 'direct' }
}
