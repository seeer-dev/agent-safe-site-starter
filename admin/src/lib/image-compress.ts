// Pure compression core — extracted from the Worker so it can be
// unit-tested with fake createImageBitmap / OffscreenCanvas.
//
// This module uses the global createImageBitmap and OffscreenCanvas
// APIs. In tests these are replaced with fakes. In the Worker they
// are the real browser APIs available in the Worker scope.
//
// GIF is NEVER passed to this module — GIF is pass/reject only.

export interface CompressResult {
  blob: Blob
  width: number
  height: number
}

export type CompressCapability =
  | { available: true }
  | { available: false; reason: string }

/**
 * Check whether the required APIs (createImageBitmap, OffscreenCanvas,
 * convertToBlob) are available in the current scope.
 */
export function checkCompressCapability(): CompressCapability {
  if (typeof globalThis.createImageBitmap === 'undefined') {
    return { available: false, reason: 'createImageBitmap not available' }
  }
  if (typeof globalThis.OffscreenCanvas === 'undefined') {
    return { available: false, reason: 'OffscreenCanvas not available' }
  }
  if (typeof globalThis.OffscreenCanvas.prototype.convertToBlob !== 'function') {
    return { available: false, reason: 'OffscreenCanvas.convertToBlob not available' }
  }
  return { available: true }
}

/**
 * Compress a JPEG or PNG blob using createImageBitmap + OffscreenCanvas.
 *
 * - JPEG: fills white background (no alpha in JPEG), outputs image/jpeg
 *   with the given quality.
 * - PNG: does NOT fill background (preserves alpha), outputs image/png.
 * - Resizes so the long edge does not exceed maxEdge.
 * - Always calls bitmap.close() in a finally block.
 *
 * Throws if the 2d context is unavailable or convertToBlob returns null.
 * The caller should check checkCompressCapability() first and classify
 * capability failures separately from compression errors.
 */
export async function compressBlob(
  blob: Blob,
  format: 'jpeg' | 'png',
  maxEdge: number,
  jpegQuality: number,
): Promise<CompressResult> {
  if (maxEdge <= 0) {
    throw new Error(`invalid maxEdge: ${maxEdge} (must be > 0)`)
  }

  const bitmap = await createImageBitmap(blob)
  try {
    const origW = bitmap.width
    const origH = bitmap.height

    if (origW <= 0 || origH <= 0) {
      throw new Error(`invalid bitmap dimensions: ${origW}x${origH}`)
    }

    const longEdge = Math.max(origW, origH)

    let targetW = origW
    let targetH = origH
    if (longEdge > maxEdge) {
      const scale = maxEdge / longEdge
      targetW = Math.round(origW * scale)
      targetH = Math.round(origH * scale)
    }

    // Clamp to at least 1px — extreme aspect ratios (e.g. 65535x1)
    // can produce targetH=0 after Math.round, which would create an
    // invalid OffscreenCanvas(2560, 0).
    if (targetW < 1) targetW = 1
    if (targetH < 1) targetH = 1

    const canvas = new OffscreenCanvas(targetW, targetH)
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      throw new Error('failed to get 2d context from OffscreenCanvas')
    }

    // For JPEG, fill white background to handle alpha → JPEG (no alpha).
    // For PNG, do NOT fill — preserve alpha channel.
    if (format === 'jpeg') {
      ctx.fillStyle = '#ffffff'
      ctx.fillRect(0, 0, targetW, targetH)
    }

    ctx.drawImage(bitmap, 0, 0, targetW, targetH)

    const outType = format === 'jpeg' ? 'image/jpeg' : 'image/png'
    const outBlob = await canvas.convertToBlob({
      type: outType,
      quality: format === 'jpeg' ? jpegQuality : undefined,
    })

    if (!outBlob) {
      throw new Error('convertToBlob returned null')
    }

    return { blob: outBlob, width: targetW, height: targetH }
  } finally {
    bitmap.close()
  }
}
