import type { z } from 'zod'
import { ApiError } from './errors'
import { apiErrorSchema } from './schemas'

export const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? '/api/v1').replace(/\/+$/, '')
export const useMocks = import.meta.env.VITE_USE_MOCKS === 'true'

const DEFAULT_TIMEOUT_MS = 30_000

interface RequestOptions {
  method?: string
  body?: unknown
  signal?: AbortSignal
  timeoutMs?: number
}

export interface UploadOptions {
  onProgress?: (fraction: number) => void
  signal?: AbortSignal
}

function parseJson(text: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    throw new ApiError('INVALID_RESPONSE', 'Ответ сервера не является JSON')
  }
}

function errorFromBody(payload: unknown, status: number): ApiError {
  const parsed = apiErrorSchema.safeParse(payload)
  if (parsed.success) {
    return new ApiError(parsed.data.error.code, parsed.data.error.message, status)
  }
  return new ApiError('INTERNAL_ERROR', `HTTP ${status}`, status)
}

function parseResponse<T>(payload: unknown, schema: z.ZodType<T>, status: number): T {
  const parsed = schema.safeParse(payload)
  if (!parsed.success) {
    throw new ApiError(
      'INVALID_RESPONSE',
      'Ответ сервера не соответствует контракту',
      status,
      parsed.error.message,
    )
  }
  return parsed.data
}

export async function request<T>(
  path: string,
  schema: z.ZodType<T>,
  options: RequestOptions = {},
): Promise<T> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? DEFAULT_TIMEOUT_MS)
  const abortFromParent = () => controller.abort()
  options.signal?.addEventListener('abort', abortFromParent)

  try {
    const response = await fetch(apiBaseUrl + path, {
      method: options.method ?? 'GET',
      headers: options.body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: controller.signal,
    })
    const text = await response.text()
    const payload = text.length > 0 ? parseJson(text) : undefined
    if (!response.ok) {
      throw errorFromBody(payload, response.status)
    }
    return parseResponse(payload, schema, response.status)
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    if (error instanceof DOMException && error.name === 'AbortError') {
      if (options.signal?.aborted) {
        throw error
      }
      throw new ApiError('TIMEOUT', 'Сервер не ответил вовремя')
    }
    throw new ApiError('NETWORK_ERROR', error instanceof Error ? error.message : 'network error')
  } finally {
    clearTimeout(timeout)
    options.signal?.removeEventListener('abort', abortFromParent)
  }
}

export function uploadFile<T>(
  path: string,
  file: File,
  schema: z.ZodType<T>,
  options: UploadOptions = {},
): Promise<T> {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    form.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', apiBaseUrl + path)
    xhr.responseType = 'text'

    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && event.total > 0) {
        options.onProgress?.(event.loaded / event.total)
      }
    }

    xhr.onload = () => {
      let payload: unknown
      try {
        payload = xhr.responseText ? JSON.parse(xhr.responseText) : undefined
      } catch {
        reject(new ApiError('INVALID_RESPONSE', 'Ответ сервера не является JSON', xhr.status))
        return
      }
      if (xhr.status < 200 || xhr.status >= 300) {
        reject(errorFromBody(payload, xhr.status))
        return
      }
      try {
        resolve(parseResponse(payload, schema, xhr.status))
      } catch (error) {
        reject(error)
      }
    }

    xhr.onerror = () => reject(new ApiError('NETWORK_ERROR', 'Не удалось загрузить файл'))
    xhr.onabort = () => reject(new DOMException('Aborted', 'AbortError'))

    if (options.signal) {
      if (options.signal.aborted) {
        reject(new DOMException('Aborted', 'AbortError'))
        return
      }
      options.signal.addEventListener('abort', () => xhr.abort(), { once: true })
    }

    xhr.send(form)
  })
}
