<template>
  <div class="media-uploader" :aria-live="ariaLive">
    <!-- Existing images list (sortable + alt edit + delete) -->
    <ul v-if="modelValue.length > 0" class="media-list">
      <li
        v-for="(img, i) in modelValue"
        :key="i"
        class="media-item"
      >
        <div class="media-item-main">
          <span class="media-key mono" :title="img.key">{{ shortKey(img.key) }}</span>
          <input
            v-if="!readOnly"
            type="text"
            class="media-alt-input"
            style="min-height:24px"
            :value="img.alt_text"
            placeholder="Alt text"
            @input="updateAlt(i, ($event.target as HTMLInputElement).value)"
            :aria-label="`Alt text for image ${i + 1}`"
          />
        </div>
        <div v-if="!readOnly" class="media-item-actions">
          <button
            v-if="i > 0"
            type="button"
            class="btn-move"
            style="min-height:24px;min-width:24px"
            @click="moveUp(i)"
            :aria-label="`Move image ${i + 1} up`"
          >↑</button>
          <button
            v-if="i < modelValue.length - 1"
            type="button"
            class="btn-move"
            style="min-height:24px;min-width:24px"
            @click="moveDown(i)"
            :aria-label="`Move image ${i + 1} down`"
          >↓</button>
          <button
            type="button"
            class="btn-remove"
            style="min-height:24px"
            @click="removeImage(i)"
            :aria-label="`Remove image ${i + 1}`"
          >移除</button>
        </div>
      </li>
    </ul>

    <!-- Upload area (hidden in read-only mode) -->
    <div v-if="!readOnly" class="media-upload-area">
      <input
        ref="fileInputRef"
        type="file"
        accept="image/jpeg,image/png,image/gif"
        class="media-file-input"
        @change="onFileSelected"
        :disabled="uploadState !== 'idle' && uploadState !== 'error' && uploadState !== 'success'"
        :aria-labelledby="labelledby || undefined"
        aria-label="Select image file"
      />
      <button
        ref="selectBtnRef"
        type="button"
        class="btn-select"
        @click="fileInputRef?.click()"
        :disabled="uploadState !== 'idle' && uploadState !== 'error' && uploadState !== 'success'"
      >選擇圖片</button>

      <!-- Status display -->
      <div v-if="uploadState !== 'idle'" class="media-status">
        <span v-if="uploadState === 'analyzing'" class="status-msg">分析中…</span>
        <span v-if="uploadState === 'compressing'" class="status-msg">壓縮中…</span>
        <span v-if="uploadState === 'uploading'" class="status-msg">上傳中…</span>
        <span v-if="uploadState === 'verifying'" class="status-msg">驗證中…</span>
        <span v-if="uploadState === 'success'" class="status-msg status-ok">已驗證，已加入列表</span>
        <span v-if="uploadState === 'error'" class="status-msg status-err">{{ uploadError }}</span>
      </div>

      <!-- Cancel button -->
      <button
        v-if="uploadState === 'analyzing' || uploadState === 'compressing' || uploadState === 'uploading' || uploadState === 'verifying'"
        type="button"
        class="btn-cancel"
        style="min-height:24px"
        @click="cancelUpload"
      >取消</button>

      <!-- Retry button (error state) -->
      <button
        v-if="uploadState === 'error' && lastFile"
        ref="retryBtnRef"
        type="button"
        class="btn-retry"
        style="min-height:24px"
        @click="retryUpload"
      >重試</button>
    </div>

    <p v-if="help" class="field-help">{{ help }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted, computed, nextTick, watch } from 'vue'
import type { ProductImageEntry, UploadState, CompressWorkerRequest, CompressWorkerResponse } from '@/lib/media-types'
import { parseImageHeaderFromFile } from '@/lib/image-header'
import {
  evaluateThresholds,
  validateCompressedOutput,
  COMPRESS_MAX_EDGE,
  JPEG_QUALITY,
} from '@/lib/media-thresholds'
import {
  presignUpload,
  uploadToR2,
  verifyUpload,
} from '@/lib/media-api'

const props = defineProps<{
  modelValue: ProductImageEntry[]
  readOnly?: boolean
  help?: string
  labelledby?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ProductImageEntry[]]
}>()

const fileInputRef = ref<HTMLInputElement | null>(null)
const selectBtnRef = ref<HTMLButtonElement | null>(null)
const retryBtnRef = ref<HTMLButtonElement | null>(null)
const uploadState = ref<UploadState>('idle')
const uploadError = ref<string>('')
const lastFile = ref<File | null>(null)

// AbortController for fetch cancellation
let abortController: AbortController | null = null
// Worker reference for termination
let compressWorker: Worker | null = null
// Success timeout ID for cleanup
let successTimeoutId: ReturnType<typeof setTimeout> | null = null
// Operation generation — incremented on every new processFile call.
// After cancellation, stale operations check their generation and
// bail out without mutating state or emitting keys.
let currentGeneration = 0

const ariaLive = computed(() => {
  if (uploadState.value === 'error') return 'assertive'
  return 'polite'
})

// Focus management: move focus to retry button on error, select button
// on idle (after success timeout). Uses nextTick to wait for DOM update.
watch(uploadState, (state) => {
  if (state === 'error') {
    nextTick(() => {
      retryBtnRef.value?.focus()
    })
  } else if (state === 'idle') {
    nextTick(() => {
      selectBtnRef.value?.focus()
    })
  }
})

function emitUpdate(images: ProductImageEntry[]) {
  emit('update:modelValue', [...images])
}

function shortKey(key: string): string {
  if (key.length <= 40) return key
  return key.slice(0, 20) + '…' + key.slice(-17)
}

function updateAlt(index: number, altText: string) {
  const images = [...props.modelValue]
  images[index] = { ...images[index], alt_text: altText }
  emitUpdate(images)
}

function removeImage(index: number) {
  const images = [...props.modelValue]
  images.splice(index, 1)
  emitUpdate(images)
}

function moveUp(index: number) {
  if (index <= 0) return
  const images = [...props.modelValue]
  const tmp = images[index - 1]
  images[index - 1] = images[index]
  images[index] = tmp
  emitUpdate(images)
}

function moveDown(index: number) {
  if (index >= props.modelValue.length - 1) return
  const images = [...props.modelValue]
  const tmp = images[index + 1]
  images[index + 1] = images[index]
  images[index] = tmp
  emitUpdate(images)
}

function clearSuccessTimeout() {
  if (successTimeoutId !== null) {
    clearTimeout(successTimeoutId)
    successTimeoutId = null
  }
}

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  // Reset input so re-selecting the same file works
  input.value = ''

  lastFile.value = file
  await processFile(file)
}

async function retryUpload() {
  if (!lastFile.value) return
  // Prevent duplicate submission — only retry from error state
  if (uploadState.value !== 'error') return
  await processFile(lastFile.value)
}

async function processFile(file: File) {
  // Guard against duplicate submission
  if (uploadState.value === 'analyzing' || uploadState.value === 'compressing' ||
      uploadState.value === 'uploading' || uploadState.value === 'verifying') {
    return
  }

  // Start a new operation generation — any previous stale operation
  // will see a mismatch and bail out.
  const gen = ++currentGeneration
  clearSuccessTimeout()

  uploadState.value = 'analyzing'
  uploadError.value = ''
  abortController = new AbortController()
  const signal = abortController.signal

  try {
    // Step 1: Parse header (main thread, minimal bytes only)
    // Header slice reads cannot be aborted, but we check signal after.
    const header = await parseImageHeaderFromFile(file)
    if (signal.aborted || gen !== currentGeneration) return

    // Step 2: Evaluate thresholds
    const decision = evaluateThresholds(file.size, header)
    if (decision.action === 'reject') {
      throw new Error(decision.reason)
    }

    let uploadBlob: Blob = file

    // Step 3: Compress if needed
    if (decision.action === 'compress') {
      if (header.format === 'gif') {
        // GIF should never reach compress — evaluateThresholds handles GIF
        // as direct/reject only. But guard anyway.
        throw new Error('GIF cannot be compressed')
      }

      uploadState.value = 'compressing'

      const workerResult = await compressInWorker(
        file,
        header.format as 'jpeg' | 'png',
        COMPRESS_MAX_EDGE,
        JPEG_QUALITY,
        signal,
      )

      // After compression, check if we were cancelled
      if (signal.aborted || gen !== currentGeneration) return

      if (workerResult.type === 'unsupported') {
        throw new Error(`compression required but Worker unavailable: ${workerResult.reason}`)
      }
      if (workerResult.type === 'error') {
        throw new Error(`compression failed: ${workerResult.message}`)
      }

      // Validate compressed output is within server limits
      const compressedDecision = validateCompressedOutput(
        workerResult.blob.size,
        workerResult.width,
        workerResult.height,
      )
      if (compressedDecision.action === 'reject') {
        throw new Error(compressedDecision.reason)
      }

      uploadBlob = workerResult.blob
    }

    // Check cancellation before presign
    if (signal.aborted || gen !== currentGeneration) return

    // Step 4: Presign
    uploadState.value = 'uploading'

    const contentType = uploadBlob.type || (header.format === 'jpeg' ? 'image/jpeg' : header.format === 'png' ? 'image/png' : 'image/gif')
    const ext = header.format === 'jpeg' ? 'jpg' : header.format === 'png' ? 'png' : 'gif'

    const presignRes = await presignUpload({
      filename: `upload.${ext}`,
      content_type: contentType,
      purpose: 'product-image',
    }, signal)

    if (signal.aborted || gen !== currentGeneration) return

    // Validate presign response shape — key must be uploads/...
    if (!presignRes.key.startsWith('uploads/')) {
      throw new Error('presign returned invalid key (expected uploads/ prefix)')
    }
    if (!presignRes.url || presignRes.method !== 'PUT') {
      throw new Error('presign returned invalid response (missing url or method != PUT)')
    }

    // Step 5: PUT to R2 (only the presigned URL, never the API base)
    await uploadToR2(presignRes.url, uploadBlob, presignRes.headers, signal)

    if (signal.aborted || gen !== currentGeneration) return

    // Step 6: Verify
    uploadState.value = 'verifying'
    const verifyRes = await verifyUpload({ key: presignRes.key }, signal)

    if (signal.aborted || gen !== currentGeneration) return

    // Validate verify response shape — key must be verified/product-images/...
    if (!verifyRes.key.startsWith('verified/product-images/')) {
      throw new Error('verify returned invalid key (expected verified/product-images/ prefix)')
    }
    if (!verifyRes.content_type || verifyRes.bytes <= 0 || verifyRes.width <= 0 || verifyRes.height <= 0) {
      throw new Error('verify returned incomplete metadata')
    }

    // Step 7: Add verified key to model — only verified keys are written.
    // Skip if the key already exists (deduplication — verify is idempotent
    // and may return the same key for a re-upload of the same content).
    const alreadyExists = props.modelValue.some((img) => img.key === verifyRes.key)
    if (!alreadyExists) {
      const newEntry: ProductImageEntry = {
        key: verifyRes.key,
        alt_text: '',
      }
      emitUpdate([...props.modelValue, newEntry])
    }
    uploadState.value = 'success'

    // Reset to idle after a brief delay so the user sees the success message
    successTimeoutId = setTimeout(() => {
      successTimeoutId = null
      if (gen === currentGeneration && uploadState.value === 'success') {
        uploadState.value = 'idle'
      }
    }, 1500)
  } catch (err) {
    // If this operation was cancelled (stale generation), do NOT
    // mutate state — the cancel handler already set the state.
    if (signal.aborted || gen !== currentGeneration) return

    if (err instanceof DOMException && err.name === 'AbortError') {
      uploadError.value = '已取消'
    } else {
      uploadError.value = err instanceof Error ? err.message : String(err)
    }
    uploadState.value = 'error'
  } finally {
    if (gen === currentGeneration) {
      abortController = null
    }
  }
}

/**
 * Compress in a Vite module Worker. Accepts an AbortSignal — on abort,
 * terminates the Worker, removes listeners, and rejects with AbortError.
 * Listeners are set BEFORE postMessage to avoid a race where the Worker
 * responds before onmessage is attached.
 */
function compressInWorker(
  blob: Blob,
  format: 'jpeg' | 'png',
  maxEdge: number,
  jpegQuality: number,
  signal: AbortSignal,
): Promise<CompressWorkerResponse> {
  return new Promise((resolve, reject) => {
    let worker: Worker

    try {
      worker = new Worker(
        new URL('../workers/image-compress.worker.ts', import.meta.url),
        { type: 'module' },
      )
      compressWorker = worker
    } catch (err) {
      // Worker construction failed — resolve as unsupported
      resolve({
        type: 'unsupported',
        reason: err instanceof Error ? err.message : 'Worker construction failed',
      })
      return
    }

    // Set up listeners BEFORE postMessage to avoid race condition
    // where the Worker responds before onmessage is attached.
    const onMessage = (ev: MessageEvent<CompressWorkerResponse>) => {
      cleanup()
      resolve(ev.data)
    }
    const onError = (ev: ErrorEvent) => {
      cleanup()
      reject(new Error(`Worker error: ${ev.message}`))
    }
    const onAbort = () => {
      cleanup()
      reject(new DOMException('Worker aborted', 'AbortError'))
    }

    function cleanup() {
      worker.removeEventListener('message', onMessage)
      worker.removeEventListener('error', onError)
      signal.removeEventListener('abort', onAbort)
      worker.terminate()
      if (compressWorker === worker) {
        compressWorker = null
      }
    }

    worker.addEventListener('message', onMessage)
    worker.addEventListener('error', onError)
    signal.addEventListener('abort', onAbort)

    const req: CompressWorkerRequest = {
      type: 'compress',
      blob,
      format,
      maxEdge,
      jpegQuality,
    }
    worker.postMessage(req)
  })
}

function cancelUpload() {
  // Increment generation so any in-flight operation bails out
  // without mutating state or emitting keys.
  currentGeneration++
  clearSuccessTimeout()

  if (abortController) {
    abortController.abort()
  }
  if (compressWorker) {
    compressWorker.terminate()
    compressWorker = null
  }
  uploadState.value = 'idle'
  uploadError.value = ''
  abortController = null
}

onUnmounted(() => {
  currentGeneration++
  clearSuccessTimeout()
  if (abortController) {
    abortController.abort()
  }
  if (compressWorker) {
    compressWorker.terminate()
    compressWorker = null
  }
})
</script>

<style scoped>
.media-uploader {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.media-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.media-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px;
  background: var(--surface-2, #f5f5f5);
  border-radius: 4px;
}
.media-item-main {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.media-key {
  font-size: 12px;
  color: var(--text-2, #666);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.media-alt-input {
  border: 1px solid var(--border, #ddd);
  border-radius: 3px;
  padding: 4px 6px;
  font-size: 12px;
  flex: 1;
  min-width: 80px;
  min-height: 24px;
}
.media-item-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.btn-move, .btn-remove, .btn-cancel, .btn-retry {
  border: 1px solid var(--border, #ddd);
  border-radius: 3px;
  padding: 4px 8px;
  font-size: 12px;
  cursor: pointer;
  background: var(--surface, #fff);
  min-height: 24px;
}
.btn-move {
  min-width: 24px;
}
.btn-remove, .btn-cancel {
  color: var(--danger, #c00);
}
.btn-retry {
  color: var(--primary, #0066cc);
}
.media-upload-area {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.media-file-input {
  display: none;
}
.btn-select {
  border: 1px solid var(--border, #ddd);
  border-radius: 4px;
  padding: 6px 12px;
  cursor: pointer;
  background: var(--surface, #fff);
  font-size: 13px;
}
.btn-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.media-status {
  display: flex;
  align-items: center;
  gap: 4px;
}
.status-msg {
  font-size: 12px;
  color: var(--text-2, #666);
}
.status-ok {
  color: var(--success, #080);
}
.status-err {
  color: var(--danger, #c00);
}
.field-help {
  font-size: 11px;
  color: var(--text-3, #999);
  margin: 0;
}
</style>
