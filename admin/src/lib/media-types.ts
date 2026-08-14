// Shared types for the media upload pipeline.

/** A product image entry in the form model. */
export interface ProductImageEntry {
  /** Verified object key from POST /api/media/verify, or existing key from admin response. */
  key: string
  /** Alt text for accessibility. Defaults to product name server-side if empty. */
  alt_text: string
}

/** Presign request body for POST /api/media/presign. */
export interface PresignRequest {
  filename: string
  content_type: string
  purpose: string
}

/** Presign response from POST /api/media/presign. */
export interface PresignResponse {
  url: string
  method: string
  key: string
  headers: Record<string, string>
}

/** Verify request body for POST /api/media/verify. */
export interface VerifyRequest {
  key: string
}

/** Verify response from POST /api/media/verify. */
export interface VerifyResponse {
  key: string
  content_type: string
  bytes: number
  width: number
  height: number
}

/** Upload pipeline states. */
export type UploadState =
  | 'idle'
  | 'analyzing'
  | 'compressing'
  | 'uploading'
  | 'verifying'
  | 'success'
  | 'error'

/** Result of header parsing — format and dimensions. */
export interface ImageHeaderInfo {
  format: 'jpeg' | 'png' | 'gif' | 'unknown'
  width: number
  height: number
}

/** Decision from threshold evaluation. */
export type ThresholdDecision =
  | { action: 'direct' }
  | { action: 'compress' }
  | { action: 'reject'; reason: string }

/** Message to the compression Worker. */
export interface CompressWorkerRequest {
  type: 'compress'
  blob: Blob
  format: 'jpeg' | 'png'
  maxEdge: number
  jpegQuality: number
}

/** Response from the compression Worker. */
export type CompressWorkerResponse =
  | { type: 'compressed'; blob: Blob; width: number; height: number }
  | { type: 'unsupported'; reason: string }
  | { type: 'error'; message: string }
