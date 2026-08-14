import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { compressBlob, checkCompressCapability } from './image-compress'
import { COMPRESS_MAX_EDGE, JPEG_QUALITY } from './media-thresholds'

// --- Fakes ---

interface FakeBitmap {
  width: number
  height: number
  close: ReturnType<typeof vi.fn>
}

interface FakeCtx {
  fillStyle: string
  fillRect: ReturnType<typeof vi.fn>
  drawImage: ReturnType<typeof vi.fn>
}

interface FakeCanvas {
  width: number
  height: number
  getContext: ReturnType<typeof vi.fn>
  convertToBlob: ReturnType<typeof vi.fn>
}

function makeFakeBitmap(w: number, h: number): FakeBitmap {
  return { width: w, height: h, close: vi.fn() }
}

function makeFakeCtx(): FakeCtx {
  return {
    fillStyle: '',
    fillRect: vi.fn(),
    drawImage: vi.fn(),
  }
}

function makeFakeCanvas(w: number, h: number, blob: Blob): FakeCanvas {
  const ctx = makeFakeCtx()
  return {
    width: w,
    height: h,
    getContext: vi.fn(() => ctx),
    convertToBlob: vi.fn(async (opts: { type: string; quality?: number }) => {
      // Return a blob with the requested type so tests can verify
      return new Blob([new Uint8Array([1, 2, 3])], { type: opts.type })
    }),
  }
}

/**
 * Create a fake OffscreenCanvas constructor with a configurable prototype.
 * Avoids `any` by using a plain function (whose prototype is writable)
 * wrapped in a typed cast.
 */
function makeOffscreenCanvasCtor(
  impl: (w: number, h: number) => FakeCanvas,
  convertToBlob: ReturnType<typeof vi.fn>,
): typeof OffscreenCanvas {
  // Plain function — its prototype is writable and configurable
  function Ctor(w: number, h: number): FakeCanvas {
    return impl(w, h)
  }
  // Plain function prototypes are writable by default
  Ctor.prototype = { convertToBlob } as unknown as OffscreenCanvas
  return Ctor as unknown as typeof OffscreenCanvas
}

function installFakes(bitmap: FakeBitmap, canvas: FakeCanvas) {
  const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
  const OffscreenCanvasCtor = makeOffscreenCanvasCtor(() => canvas, canvas.convertToBlob)

  globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
  globalThis.OffscreenCanvas = OffscreenCanvasCtor
}

function restoreGlobals() {
  // @ts-expect-error — restoring to undefined for test isolation
  delete globalThis.createImageBitmap
  // @ts-expect-error — restoring to undefined for test isolation
  delete globalThis.OffscreenCanvas
}

describe('checkCompressCapability', () => {
  afterEach(() => restoreGlobals())

  it('returns unavailable when createImageBitmap is missing', () => {
    restoreGlobals()
    const cap = checkCompressCapability()
    expect(cap.available).toBe(false)
    if (!cap.available) expect(cap.reason).toContain('createImageBitmap')
  })

  it('returns unavailable when OffscreenCanvas is missing', () => {
    globalThis.createImageBitmap = vi.fn() as unknown as typeof globalThis.createImageBitmap
    // @ts-expect-error — OffscreenCanvas not defined
    delete globalThis.OffscreenCanvas
    const cap = checkCompressCapability()
    expect(cap.available).toBe(false)
    if (!cap.available) expect(cap.reason).toContain('OffscreenCanvas')
  })

  it('returns unavailable when convertToBlob is missing', () => {
    globalThis.createImageBitmap = vi.fn() as unknown as typeof globalThis.createImageBitmap
    // No convertToBlob on prototype — plain function with empty prototype
    function EmptyCtor() {}
    globalThis.OffscreenCanvas = EmptyCtor as unknown as typeof OffscreenCanvas
    const cap = checkCompressCapability()
    expect(cap.available).toBe(false)
    if (!cap.available) expect(cap.reason).toContain('convertToBlob')
  })

  it('returns available when all APIs present', () => {
    globalThis.createImageBitmap = vi.fn() as unknown as typeof globalThis.createImageBitmap
    const FakeOC = makeOffscreenCanvasCtor(() => ({} as FakeCanvas), vi.fn())
    globalThis.OffscreenCanvas = FakeOC
    const cap = checkCompressCapability()
    expect(cap.available).toBe(true)
  })
})

describe('compressBlob', () => {
  let bitmap: FakeBitmap
  let canvas: FakeCanvas
  let ctx: FakeCtx

  beforeEach(() => {
    bitmap = makeFakeBitmap(4000, 3000)
    ctx = makeFakeCtx()
    canvas = {
      width: 0,
      height: 0,
      getContext: vi.fn(() => ctx),
      convertToBlob: vi.fn(async (opts: { type: string; quality?: number }) => {
        return new Blob([new Uint8Array([1, 2, 3])], { type: opts.type })
      }),
    }
    // Override installFakes to use our pre-made ctx and canvas
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor
  })

  afterEach(() => restoreGlobals())

  it('JPEG: fills white background, outputs image/jpeg with quality .82, resizes to <=2560', async () => {
    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    const result = await compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    // White fill was applied
    expect(ctx.fillStyle).toBe('#ffffff')
    expect(ctx.fillRect).toHaveBeenCalledWith(0, 0, expect.any(Number), expect.any(Number))

    // drawImage was called with the bitmap
    expect(ctx.drawImage).toHaveBeenCalledWith(bitmap, 0, 0, expect.any(Number), expect.any(Number))

    // convertToBlob was called with image/jpeg and quality .82
    expect(canvas.convertToBlob).toHaveBeenCalledWith({
      type: 'image/jpeg',
      quality: JPEG_QUALITY,
    })

    // Output blob type is image/jpeg
    expect(result.blob.type).toBe('image/jpeg')

    // Resized: long edge <= 2560
    expect(result.width).toBeLessThanOrEqual(COMPRESS_MAX_EDGE)
    expect(result.height).toBeLessThanOrEqual(COMPRESS_MAX_EDGE)
    // 4000x3000 -> scale 2560/4000 = 0.64 -> 2560x1920
    expect(result.width).toBe(2560)
    expect(result.height).toBe(1920)

    // bitmap.close was called
    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('PNG: does NOT fill background, outputs image/png, preserves alpha', async () => {
    const blob = new Blob([new Uint8Array(100)], { type: 'image/png' })
    const result = await compressBlob(blob, 'png', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    // No fillRect for PNG (alpha preserved)
    expect(ctx.fillRect).not.toHaveBeenCalled()
    expect(ctx.fillStyle).toBe('')

    // drawImage was called
    expect(ctx.drawImage).toHaveBeenCalledWith(bitmap, 0, 0, expect.any(Number), expect.any(Number))

    // convertToBlob was called with image/png and NO quality
    expect(canvas.convertToBlob).toHaveBeenCalledWith({
      type: 'image/png',
      quality: undefined,
    })

    // Output blob type is image/png
    expect(result.blob.type).toBe('image/png')

    // bitmap.close was called
    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('does not resize when image is already <= maxEdge', async () => {
    bitmap = makeFakeBitmap(1920, 1080)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    const result = await compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    // No resize — original dimensions preserved
    expect(result.width).toBe(1920)
    expect(result.height).toBe(1080)
  })

  it('always calls bitmap.close even on error', async () => {
    // Make convertToBlob throw
    canvas.convertToBlob = vi.fn(async () => {
      throw new Error('convertToBlob failed')
    })

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    await expect(compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)).rejects.toThrow('convertToBlob failed')

    // bitmap.close was still called
    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('throws when 2d context is unavailable', async () => {
    canvas.getContext = vi.fn(() => null)

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    await expect(compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)).rejects.toThrow('2d context')

    // bitmap.close was still called (finally block)
    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('throws when convertToBlob returns null', async () => {
    canvas.convertToBlob = vi.fn(async () => null)

    const blob = new Blob([new Uint8Array(100)], { type: 'image/png' })
    await expect(compressBlob(blob, 'png', COMPRESS_MAX_EDGE, JPEG_QUALITY)).rejects.toThrow('null')

    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('resize maintains aspect ratio', async () => {
    // 8000x2000 -> long edge 8000 -> scale 2560/8000 = 0.32 -> 2560x640
    bitmap = makeFakeBitmap(8000, 2000)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    const result = await compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    expect(result.width).toBe(2560)
    expect(result.height).toBe(640)
    // Aspect ratio preserved: 8000/2000 = 4, 2560/640 = 4
    expect(result.width / result.height).toBeCloseTo(4, 1)
  })

  // --- Extreme aspect ratio clamping ---

  it('extreme aspect ratio (65535x1): clamps targetH to 1, no zero-dim canvas', async () => {
    // 65535x1 — within 64 MP hard limit (65535 pixels). After scaling
    // to maxEdge=2560: scale = 2560/65535 ≈ 0.0390625
    // targetW = round(65535 * 0.0390625) = 2560
    // targetH = round(1 * 0.0390625) = round(0.039) = 0 → clamped to 1
    bitmap = makeFakeBitmap(65535, 1)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    const result = await compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    // targetW should be 2560 (long edge scaled)
    expect(result.width).toBe(2560)
    // targetH should be clamped to 1, not 0
    expect(result.height).toBe(1)

    // Canvas should have been created with (2560, 1), not (2560, 0)
    // — verified via the impl recording width/height on the canvas object
    expect(canvas.width).toBe(2560)
    expect(canvas.height).toBe(1)
  })

  it('extreme aspect ratio (1x65535): clamps targetW to 1', async () => {
    // 1x65535 — long edge is height. After scaling:
    // targetW = round(1 * 0.039) = 0 → clamped to 1
    // targetH = round(65535 * 0.039) = 2560
    bitmap = makeFakeBitmap(1, 65535)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/png' })
    const result = await compressBlob(blob, 'png', COMPRESS_MAX_EDGE, JPEG_QUALITY)

    expect(result.width).toBe(1)
    expect(result.height).toBe(2560)
    expect(canvas.width).toBe(1)
    expect(canvas.height).toBe(2560)
  })

  // --- Zero/negative dimension guards ---

  it('throws when bitmap width is 0', async () => {
    bitmap = makeFakeBitmap(0, 100)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    await expect(compressBlob(blob, 'jpeg', COMPRESS_MAX_EDGE, JPEG_QUALITY)).rejects.toThrow('invalid bitmap dimensions')

    // bitmap.close still called (finally block)
    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('throws when bitmap height is 0', async () => {
    bitmap = makeFakeBitmap(100, 0)
    const createImageBitmapSpy = vi.fn(async (_src: Blob) => bitmap)
    const OffscreenCanvasCtor = makeOffscreenCanvasCtor((w, h) => {
      canvas.width = w
      canvas.height = h
      return canvas
    }, canvas.convertToBlob)
    globalThis.createImageBitmap = createImageBitmapSpy as unknown as typeof globalThis.createImageBitmap
    globalThis.OffscreenCanvas = OffscreenCanvasCtor

    const blob = new Blob([new Uint8Array(100)], { type: 'image/png' })
    await expect(compressBlob(blob, 'png', COMPRESS_MAX_EDGE, JPEG_QUALITY)).rejects.toThrow('invalid bitmap dimensions')

    expect(bitmap.close).toHaveBeenCalledTimes(1)
  })

  it('throws when maxEdge is 0', async () => {
    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    await expect(compressBlob(blob, 'jpeg', 0, JPEG_QUALITY)).rejects.toThrow('invalid maxEdge')
  })

  it('throws when maxEdge is negative', async () => {
    const blob = new Blob([new Uint8Array(100)], { type: 'image/jpeg' })
    await expect(compressBlob(blob, 'jpeg', -100, JPEG_QUALITY)).rejects.toThrow('invalid maxEdge')
  })
})
