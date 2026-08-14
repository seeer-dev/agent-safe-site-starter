import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import MediaUploader from './MediaUploader.vue'
import type { ProductImageEntry, VerifyResponse, CompressWorkerRequest, CompressWorkerResponse } from '@/lib/media-types'
import { COMPRESS_MAX_EDGE, JPEG_QUALITY } from '@/lib/media-thresholds'

// Mock the media-api module
vi.mock('@/lib/media-api', () => ({
  presignUpload: vi.fn(),
  uploadToR2: vi.fn(),
  verifyUpload: vi.fn(),
}))

// Mock the image-header module
vi.mock('@/lib/image-header', () => ({
  parseImageHeaderFromFile: vi.fn(),
}))

// Import mocked modules for per-test control
import { presignUpload, uploadToR2, verifyUpload } from '@/lib/media-api'
import { parseImageHeaderFromFile } from '@/lib/image-header'

const mockedPresignUpload = vi.mocked(presignUpload)
const mockedUploadToR2 = vi.mocked(uploadToR2)
const mockedVerifyUpload = vi.mocked(verifyUpload)
const mockedParseHeader = vi.mocked(parseImageHeaderFromFile)

function makeFile(name: string, size: number, type: string): File {
  const blob = new Blob([new Uint8Array(size)], { type })
  return new File([blob], name, { type })
}

/** Set files on an input element (re-definable for multi-upload tests).
 *  Accepts any wrapper that exposes an `.element` property (DOMWrapper
 *  or the Omit variant returned by wrapper.get()). */
function setInputFiles(input: { element: Element }, file: File) {
  Object.defineProperty(input.element as HTMLInputElement, 'files', {
    value: [file],
    writable: true,
    configurable: true,
  })
}

// --- Fake Worker infrastructure ---

interface FakeWorker extends EventTarget {
  postMessage: ReturnType<typeof vi.fn>
  terminate: ReturnType<typeof vi.fn>
  /** Deliver a message to the worker's onmessage listener. */
  _respond: (resp: CompressWorkerResponse) => void
}

let lastFakeWorker: FakeWorker | null = null
let workerCtorSpy: ReturnType<typeof vi.fn> | null = null
let originalWorker: typeof globalThis.Worker | undefined

function installFakeWorker() {
  originalWorker = globalThis.Worker
  workerCtorSpy = vi.fn()
  globalThis.Worker = workerCtorSpy as unknown as typeof globalThis.Worker

  workerCtorSpy.mockImplementation(() => {
    const target = new EventTarget() as FakeWorker
    target.postMessage = vi.fn()
    target.terminate = vi.fn()
    target._respond = (resp: CompressWorkerResponse) => {
      target.dispatchEvent(new MessageEvent('message', { data: resp }))
    }
    lastFakeWorker = target
    return target
  })
}

function restoreWorker() {
  if (originalWorker !== undefined) {
    globalThis.Worker = originalWorker
  } else {
    // @ts-expect-error — Worker was not defined before
    delete globalThis.Worker
  }
  lastFakeWorker = null
  workerCtorSpy = null
}

// ---

function makeWrapper(images: ProductImageEntry[] = []) {
  return mount(MediaUploader, {
    props: {
      modelValue: images,
    },
    attachTo: document.body,
  })
}

/** Standard valid verify response for reuse in tests. */
const VALID_VERIFY: VerifyResponse = {
  key: 'verified/product-images/user1/sha256.jpg',
  content_type: 'image/jpeg',
  bytes: 400000,
  width: 1920,
  height: 1080,
}

/** Standard valid presign response for reuse in tests. */
const VALID_PRESIGN = {
  url: 'https://r2.example.com/upload',
  method: 'PUT',
  key: 'uploads/user1/product-images/abc.jpg',
  headers: { 'Content-Type': 'image/jpeg' },
}

/** Set up mocks for a successful direct-upload flow. */
function mockSuccessfulDirectUpload() {
  mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
  mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
  mockedUploadToR2.mockResolvedValue(undefined)
  mockedVerifyUpload.mockResolvedValue(VALID_VERIFY)
}

describe('MediaUploader', () => {
  enableAutoUnmount(afterEach)

  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    restoreWorker()
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  // --- Rendering ---

  it('renders existing images from modelValue', () => {
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/abc.jpg', alt_text: 'test' },
    ]
    const wrapper = makeWrapper(images)
    expect(wrapper.text()).toContain('verified/product-images/u/abc.jpg')
  })

  it('shows read-only view when readOnly prop is true', () => {
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/a.jpg', alt_text: 'test' },
    ]
    const wrapper = mount(MediaUploader, {
      props: { modelValue: images, readOnly: true },
    })
    // No upload area, no remove buttons
    expect(wrapper.find('.media-upload-area').exists()).toBe(false)
    expect(wrapper.find('.btn-remove').exists()).toBe(false)
  })

  // --- List manipulation ---

  it('emits update:modelValue when removing an image', async () => {
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/a.jpg', alt_text: '' },
      { key: 'verified/product-images/u/b.jpg', alt_text: '' },
    ]
    const wrapper = makeWrapper(images)
    const removeButtons = wrapper.findAll('.btn-remove')
    await removeButtons[0].trigger('click')
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toHaveLength(1)
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe('verified/product-images/u/b.jpg')
  })

  it('emits update:modelValue with updated alt_text', async () => {
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/a.jpg', alt_text: '' },
    ]
    const wrapper = makeWrapper(images)
    const input = wrapper.find('.media-alt-input')
    await input.setValue('new alt')
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].alt_text).toBe('new alt')
  })

  it('moves images up and down', async () => {
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/a.jpg', alt_text: '' },
      { key: 'verified/product-images/u/b.jpg', alt_text: '' },
    ]
    const wrapper = makeWrapper(images)
    // Second item has an ↑ button (first item does not since i=0)
    const moveUpButtons = wrapper.findAll('.btn-move').filter(b => b.text().includes('↑'))
    expect(moveUpButtons).toHaveLength(1)
    await moveUpButtons[0].trigger('click')
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe('verified/product-images/u/b.jpg')
  })

  // --- Full upload flow (direct, no compression) ---

  it('completes full upload flow: presign -> PUT -> verify -> emit key', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg') // 500 KiB, direct
    mockSuccessfulDirectUpload()

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    expect(mockedPresignUpload).toHaveBeenCalledWith(
      expect.objectContaining({
        filename: 'upload.jpg',
        content_type: 'image/jpeg',
        purpose: 'product-image',
      }),
      expect.any(AbortSignal),
    )
    expect(mockedUploadToR2).toHaveBeenCalledWith(
      'https://r2.example.com/upload',
      expect.any(Blob),
      { 'Content-Type': 'image/jpeg' },
      expect.any(AbortSignal),
    )
    expect(mockedVerifyUpload).toHaveBeenCalledWith(
      { key: 'uploads/user1/product-images/abc.jpg' },
      expect.any(AbortSignal),
    )
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe('verified/product-images/user1/sha256.jpg')
  })

  // --- Compression via fake Worker ---

  it('large JPEG: compresses via Worker, sends compressed blob to presign/PUT', async () => {
    installFakeWorker()
    // 3 MiB JPEG, 4000x3000 — triggers compression (>2MiB AND >2560px)
    const file = makeFile('large.jpg', 3 * 1024 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 4000, height: 3000 })

    // Compressed blob that the fake Worker will return
    const compressedBlob = new Blob([new Uint8Array(500 * 1024)], { type: 'image/jpeg' })

    mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue(VALID_VERIFY)

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises() // past header parse, now in compressing state

    // Worker should have been constructed and postMessage called
    expect(workerCtorSpy).toHaveBeenCalledTimes(1)
    expect(lastFakeWorker).not.toBeNull()
    expect(lastFakeWorker!.postMessage).toHaveBeenCalledTimes(1)

    // Verify the request sent to the Worker
    const workerReq = lastFakeWorker!.postMessage.mock.calls[0][0] as CompressWorkerRequest
    expect(workerReq.type).toBe('compress')
    expect(workerReq.format).toBe('jpeg')
    expect(workerReq.maxEdge).toBe(COMPRESS_MAX_EDGE)
    expect(workerReq.jpegQuality).toBe(JPEG_QUALITY)

    // Deliver compressed response from the Worker
    lastFakeWorker!._respond({
      type: 'compressed',
      blob: compressedBlob,
      width: 2560,
      height: 1920,
    })
    await flushPromises()

    // Presign should be called with the compressed blob's content type
    expect(mockedPresignUpload).toHaveBeenCalledWith(
      expect.objectContaining({
        content_type: 'image/jpeg',
        filename: 'upload.jpg',
      }),
      expect.any(AbortSignal),
    )

    // PUT should receive the compressed blob, not the original file
    expect(mockedUploadToR2).toHaveBeenCalledWith(
      'https://r2.example.com/upload',
      compressedBlob,
      { 'Content-Type': 'image/jpeg' },
      expect.any(AbortSignal),
    )

    // Verify called with the presign key
    expect(mockedVerifyUpload).toHaveBeenCalledWith(
      { key: 'uploads/user1/product-images/abc.jpg' },
      expect.any(AbortSignal),
    )

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe(VALID_VERIFY.key)
  })

  it('large PNG: compresses via Worker, PUT content-type is image/png', async () => {
    installFakeWorker()
    // 3 MiB PNG, 4000x3000 — triggers compression
    const file = makeFile('large.png', 3 * 1024 * 1024, 'image/png')
    mockedParseHeader.mockResolvedValue({ format: 'png', width: 4000, height: 3000 })

    const compressedBlob = new Blob([new Uint8Array(500 * 1024)], { type: 'image/png' })

    mockedPresignUpload.mockResolvedValue({
      ...VALID_PRESIGN,
      key: 'uploads/user1/product-images/abc.png',
      headers: { 'Content-Type': 'image/png' },
    })
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue({
      ...VALID_VERIFY,
      key: 'verified/product-images/user1/sha256.png',
      content_type: 'image/png',
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Verify Worker request format is png
    expect(lastFakeWorker).not.toBeNull()
    const workerReq = lastFakeWorker!.postMessage.mock.calls[0][0] as CompressWorkerRequest
    expect(workerReq.format).toBe('png')
    expect(workerReq.maxEdge).toBe(COMPRESS_MAX_EDGE)

    // Deliver compressed response
    lastFakeWorker!._respond({
      type: 'compressed',
      blob: compressedBlob,
      width: 2560,
      height: 1920,
    })
    await flushPromises()

    // Presign should use image/png content type and .png extension
    expect(mockedPresignUpload).toHaveBeenCalledWith(
      expect.objectContaining({
        content_type: 'image/png',
        filename: 'upload.png',
      }),
      expect.any(AbortSignal),
    )

    // PUT should receive the compressed blob with image/png content type
    expect(mockedUploadToR2).toHaveBeenCalledWith(
      'https://r2.example.com/upload',
      compressedBlob,
      { 'Content-Type': 'image/png' },
      expect.any(AbortSignal),
    )

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe('verified/product-images/user1/sha256.png')
  })

  it('fake Worker returns unsupported: no presign, no emit, can retry', async () => {
    installFakeWorker()
    // 3 MiB JPEG — triggers compression
    const file = makeFile('large.jpg', 3 * 1024 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 4000, height: 3000 })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Worker should have been constructed
    expect(lastFakeWorker).not.toBeNull()

    // Deliver unsupported response
    lastFakeWorker!._respond({
      type: 'unsupported',
      reason: 'OffscreenCanvas not available',
    })
    await flushPromises()

    // No presign, no upload, no verify
    expect(mockedPresignUpload).not.toHaveBeenCalled()
    expect(mockedUploadToR2).not.toHaveBeenCalled()
    expect(mockedVerifyUpload).not.toHaveBeenCalled()

    // No key emitted
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()

    // Error state — should show error and retry button
    expect(wrapper.text()).toContain('Worker unavailable')
    expect(wrapper.find('.btn-retry').exists()).toBe(true)

    // Now retry — set up for success this time
    // Use a small file that goes direct (no compression needed)
    const smallFile = makeFile('small.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue(VALID_VERIFY)

    // Click retry — it should re-process the same file
    // But retry re-processes the lastFile which is the large file...
    // So instead, just select a new small file
    setInputFiles(input, smallFile)
    await input.trigger('change')
    await flushPromises()

    // Should succeed now
    expect(mockedPresignUpload).toHaveBeenCalledTimes(1)
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe(VALID_VERIFY.key)
  })

  it('cancel during compress: Worker terminated, no late emit', async () => {
    installFakeWorker()
    // 3 MiB JPEG — triggers compression
    const file = makeFile('large.jpg', 3 * 1024 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 4000, height: 3000 })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises() // now in compressing state

    // Worker should exist and be waiting for a response
    expect(lastFakeWorker).not.toBeNull()
    const worker = lastFakeWorker!
    expect(worker.postMessage).toHaveBeenCalledTimes(1)

    // Cancel while compressing
    const cancelButton = wrapper.get('.btn-cancel')
    await cancelButton.trigger('click')
    await flushPromises()

    // Worker should have been terminated (cancelUpload calls terminate)
    expect(worker.terminate).toHaveBeenCalled()

    // Now late-respond from the Worker (simulating race) — should NOT emit
    worker._respond({
      type: 'compressed',
      blob: new Blob([new Uint8Array(100)], { type: 'image/jpeg' }),
      width: 2560,
      height: 1920,
    })
    await flushPromises()

    // No presign, no emit
    expect(mockedPresignUpload).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()

    // State should be idle
    expect(wrapper.find('.btn-cancel').exists()).toBe(false)
    expect(wrapper.find('.status-msg').exists()).toBe(false)
    const selectBtn = wrapper.get('.btn-select')
    expect(selectBtn.attributes('disabled')).toBeFalsy()
    // Focus should move to the select button after cancel returns to idle
    await nextTick()
    expect(selectBtn.element).toBe(document.activeElement)
  })

  // --- Response validation ---

  it('rejects presign response with invalid key (not uploads/)', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockResolvedValue({
      ...VALID_PRESIGN,
      key: 'invalid/key.jpg', // not uploads/
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy() // no key emitted
    expect(wrapper.text()).toContain('presign returned invalid key')
  })

  it('rejects verify response with invalid key (not verified/product-images/)', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue({
      ...VALID_VERIFY,
      key: 'invalid/key.jpg', // not verified/product-images/
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
    expect(wrapper.text()).toContain('verify returned invalid key')
  })

  it('rejects verify response with incomplete metadata', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue({
      key: 'verified/product-images/user1/sha256.jpg',
      content_type: '', // empty
      bytes: 0,
      width: 0,
      height: 0,
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
    expect(wrapper.text()).toContain('incomplete metadata')
  })

  // --- Deduplication ---

  it('does not append duplicate when verify returns key already in model', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockSuccessfulDirectUpload()

    // Pre-populate model with the same key that verify will return
    const existing: ProductImageEntry[] = [
      { key: VALID_VERIFY.key, alt_text: 'existing' },
    ]
    const wrapper = makeWrapper(existing)
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Should show success (verified key is valid)
    expect(wrapper.text()).toContain('已驗證')
    // Should NOT emit a duplicate — no update:modelValue event
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
  })

  // --- Cancellation (direct upload path) ---

  it('cancel during presign prevents key emission and returns to idle', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })

    // Make presign hang forever (never resolve) so we can cancel mid-flight
    mockedPresignUpload.mockImplementation((_req, signal) => {
      return new Promise((_resolve, reject) => {
        if (signal) {
          signal.addEventListener('abort', () => {
            reject(new DOMException('aborted', 'AbortError'))
          })
        }
      })
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises() // let header parse + threshold evaluate

    // Cancel while presign is pending
    const cancelButton = wrapper.get('.btn-cancel')
    await cancelButton.trigger('click')
    await flushPromises()

    // State should be idle: no status message visible, no cancel button,
    // and the select button should be enabled (not disabled).
    expect(wrapper.find('.btn-cancel').exists()).toBe(false)
    expect(wrapper.find('.status-msg').exists()).toBe(false)
    const selectBtn = wrapper.get('.btn-select')
    expect(selectBtn.attributes('disabled')).toBeFalsy()
    // Focus should move to the select button after cancel returns to idle
    await nextTick()
    expect(selectBtn.element).toBe(document.activeElement)
    // No key emitted
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
  })

  it('cancel during verify prevents late verify from emitting key', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockResolvedValue(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)

    // Make verify hang, then resolve late after cancel
    let verifyResolve!: (v: VerifyResponse) => void
    mockedVerifyUpload.mockImplementation((_req, signal) => {
      return new Promise<VerifyResponse>((resolve, reject) => {
        verifyResolve = resolve
        if (signal) {
          signal.addEventListener('abort', () => {
            reject(new DOMException('aborted', 'AbortError'))
          })
        }
      })
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises() // past presign + upload, now in verify

    // Cancel while verify is pending
    const cancelButton = wrapper.get('.btn-cancel')
    await cancelButton.trigger('click')
    await flushPromises()

    // Now late-resolve verify (simulating race) — should NOT emit
    verifyResolve(VALID_VERIFY)
    await flushPromises()

    // No key should have been emitted
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
    // State should be idle: no cancel button, select enabled
    expect(wrapper.find('.btn-cancel').exists()).toBe(false)
    const selectBtn = wrapper.get('.btn-select')
    expect(selectBtn.attributes('disabled')).toBeFalsy()
  })

  it('unmount cleanup aborts in-flight operations', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })

    // Make presign hang
    mockedPresignUpload.mockImplementation((_req, signal) => {
      return new Promise((_resolve, reject) => {
        if (signal) {
          signal.addEventListener('abort', () => {
            reject(new DOMException('aborted', 'AbortError'))
          })
        }
      })
    })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Unmount while presign is pending
    wrapper.unmount()
    await flushPromises()

    // No key emitted, no error thrown
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
  })

  // --- Success timeout cleanup ---

  it('clears success timeout on unmount', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockSuccessfulDirectUpload()

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Should be in success state — success message visible
    expect(wrapper.text()).toContain('已驗證')

    // Unmount before timeout fires — should not throw
    wrapper.unmount()

    // Advance timers — should not cause errors
    vi.advanceTimersByTime(2000)
  })

  it('clears success timeout on cancel', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockSuccessfulDirectUpload()

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Should be in success state
    expect(wrapper.text()).toContain('已驗證')

    // Start a new upload (which cancels the success timeout)
    const file2 = makeFile('test2.jpg', 500 * 1024, 'image/jpeg')
    setInputFiles(input, file2)
    await input.trigger('change')
    await flushPromises()

    // Advance timers — the old success timeout should have been cleared
    // and should not fire spuriously
    vi.advanceTimersByTime(2000)
  })

  it('success state transitions to idle after timeout', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockSuccessfulDirectUpload()

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Should be in success state
    expect(wrapper.text()).toContain('已驗證')

    // Advance past the 1.5s success timeout
    vi.advanceTimersByTime(1600)
    await flushPromises()

    // Should be back to idle — no success message
    expect(wrapper.text()).not.toContain('已驗證')
  })

  // --- Threshold rejection ---

  it('does not emit key when threshold rejects the file', async () => {
    const file = makeFile('test.jpg', 51 * 1024 * 1024, 'image/jpeg') // > 50 MiB
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1000, height: 1000 })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeFalsy()
    expect(wrapper.text()).toContain('50 MiB')
  })

  // --- Retry from error -> success ---

  it('retry from error succeeds and emits exactly once', async () => {
    // First attempt: presign fails
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1920, height: 1080 })
    mockedPresignUpload.mockRejectedValueOnce(new Error('network error'))

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Should be in error state
    expect(wrapper.find('.btn-retry').exists()).toBe(true)
    expect(wrapper.text()).toContain('network error')
    // No emit yet
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()

    // Set up for success on retry
    mockedPresignUpload.mockResolvedValueOnce(VALID_PRESIGN)
    mockedUploadToR2.mockResolvedValue(undefined)
    mockedVerifyUpload.mockResolvedValue(VALID_VERIFY)

    // Click retry
    await wrapper.get('.btn-retry').trigger('click')
    await flushPromises()

    // Should have succeeded
    expect(wrapper.text()).toContain('已驗證')
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    // Exactly one emit
    expect(emitted!).toHaveLength(1)
    expect((emitted![0][0] as ProductImageEntry[])[0].key).toBe(VALID_VERIFY.key)
  })

  // --- Focus management ---

  it('moves focus to retry button after error', async () => {
    const file = makeFile('test.jpg', 51 * 1024 * 1024, 'image/jpeg') // > 50 MiB
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1000, height: 1000 })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Error state — retry button should exist and be focused
    await nextTick()
    const retryBtn = wrapper.find('.btn-retry')
    expect(retryBtn.exists()).toBe(true)
    expect(retryBtn.element).toBe(document.activeElement)
  })

  it('retry from error re-enters error state and refocuses retry button', async () => {
    const file = makeFile('test.jpg', 51 * 1024 * 1024, 'image/jpeg') // > 50 MiB
    mockedParseHeader.mockResolvedValue({ format: 'jpeg', width: 1000, height: 1000 })

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Error state
    expect(wrapper.find('.btn-retry').exists()).toBe(true)

    // Retry — which will fail again with the same error
    await wrapper.get('.btn-retry').trigger('click')
    await flushPromises()

    // Should be back in error state with retry focused
    await nextTick()
    const retryBtn = wrapper.find('.btn-retry')
    expect(retryBtn.exists()).toBe(true)
    expect(retryBtn.element).toBe(document.activeElement)
  })

  it('moves focus to select button after success timeout', async () => {
    const file = makeFile('test.jpg', 500 * 1024, 'image/jpeg')
    mockSuccessfulDirectUpload()

    const wrapper = makeWrapper([])
    const input = wrapper.get('.media-file-input')
    setInputFiles(input, file)
    await input.trigger('change')
    await flushPromises()

    // Success state
    expect(wrapper.text()).toContain('已驗證')

    // Advance past the 1.5s success timeout
    vi.advanceTimersByTime(1600)
    await flushPromises()

    // Should be idle — select button should be focused
    await nextTick()
    const selectBtn = wrapper.get('.btn-select')
    expect(selectBtn.element).toBe(document.activeElement)
  })

  // --- Touch-target size (WCAG 2.5.8 AA: 24x24 minimum) ---

  it('media controls meet 24x24 minimum touch-target size', () => {
    // Mount >=2 images so the .btn-move (up/down) buttons render.
    const images: ProductImageEntry[] = [
      { key: 'verified/product-images/u/a.jpg', alt_text: '' },
      { key: 'verified/product-images/u/b.jpg', alt_text: '' },
    ]
    const wrapper = makeWrapper(images)

    // Assert inline min-height/min-width on the rendered small controls.
    // Inline style is used (alongside the scoped CSS rule) so the
    // constraint is directly observable in jsdom without a layout
    // engine or style-tag parsing.
    const moveBtns = wrapper.findAll('.btn-move')
    expect(moveBtns.length).toBeGreaterThanOrEqual(1)
    for (const btn of moveBtns) {
      expect((btn.element as HTMLElement).style.minHeight).toBe('24px')
      expect((btn.element as HTMLElement).style.minWidth).toBe('24px')
    }
    const removeBtn = wrapper.find('.btn-remove')
    expect((removeBtn.element as HTMLElement).style.minHeight).toBe('24px')
    // alt-text input only needs height (its width already exceeds 24)
    const altInput = wrapper.find('.media-alt-input')
    expect((altInput.element as HTMLInputElement).style.minHeight).toBe('24px')
  })

  // --- Label association via aria-labelledby ---

  it('file input has aria-labelledby when labelledby prop is set', () => {
    const wrapper = mount(MediaUploader, {
      props: {
        modelValue: [],
        labelledby: 'label-field-product_images',
      },
    })
    const fileInput = wrapper.find('.media-file-input')
    expect(fileInput.attributes('aria-labelledby')).toBe('label-field-product_images')
  })

  it('file input has no aria-labelledby when labelledby prop is absent', () => {
    const wrapper = makeWrapper([])
    const fileInput = wrapper.find('.media-file-input')
    expect(fileInput.attributes('aria-labelledby')).toBeFalsy()
  })
})
