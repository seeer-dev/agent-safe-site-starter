// Image compression Worker — protocol glue only.
// The actual compression logic lives in admin/src/lib/image-compress.ts
// so it can be unit-tested with fake createImageBitmap / OffscreenCanvas.
//
// GIF is NEVER sent to this Worker — GIF is pass/reject only.

import type { CompressWorkerRequest, CompressWorkerResponse } from '../lib/media-types'
import { compressBlob, checkCompressCapability } from '../lib/image-compress'

// Narrow Worker scope type — avoids `any` while staying compatible with
// both the Vite module Worker context and the TS lib types.
type WorkerScope = {
  postMessage: (msg: CompressWorkerResponse) => void
  onmessage: ((ev: MessageEvent<CompressWorkerRequest>) => void) | null
}

const scope = self as unknown as WorkerScope

scope.onmessage = (ev: MessageEvent<CompressWorkerRequest>) => {
  const req = ev.data
  if (req.type !== 'compress') return

  const cap = checkCompressCapability()
  if (!cap.available) {
    const resp: CompressWorkerResponse = {
      type: 'unsupported',
      reason: cap.reason,
    }
    scope.postMessage(resp)
    return
  }

  compressBlob(req.blob, req.format, req.maxEdge, req.jpegQuality)
    .then((result) => {
      const resp: CompressWorkerResponse = {
        type: 'compressed',
        blob: result.blob,
        width: result.width,
        height: result.height,
      }
      scope.postMessage(resp)
    })
    .catch((err) => {
      const resp: CompressWorkerResponse = {
        type: 'error',
        message: err instanceof Error ? err.message : String(err),
      }
      scope.postMessage(resp)
    })
}
