import { useCallback, useEffect, useRef, useState } from 'react'
import { apiBaseUrl } from '../client'
import { indexDoneEventSchema, indexErrorEventSchema, indexProgressEventSchema } from '../schemas'
import type { IndexDoneEvent, IndexErrorEvent, IndexProgressEvent } from '../schemas'
import { useIndexStatus } from './useIndex'

export interface IndexEventHandlers {
  onProgress?: (event: IndexProgressEvent) => void
  onDone?: (event: IndexDoneEvent) => void
  onError?: (event: IndexErrorEvent) => void
  onDisconnect?: () => void
}

export function useIndexEventStream(enabled: boolean, handlers: IndexEventHandlers) {
  const handlersRef = useRef(handlers)

  useEffect(() => {
    handlersRef.current = handlers
  }, [handlers])

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') {
      return
    }

    const source = new EventSource(`${apiBaseUrl}/index/events`)
    let closed = false

    const payloadOf = (event: Event): unknown => {
      const message = event as MessageEvent<string>
      try {
        return JSON.parse(message.data)
      } catch {
        return undefined
      }
    }

    const handleProgress = (event: Event) => {
      const parsed = indexProgressEventSchema.safeParse(payloadOf(event))
      if (parsed.success) {
        handlersRef.current.onProgress?.(parsed.data)
      }
    }

    const handleDone = (event: Event) => {
      const parsed = indexDoneEventSchema.safeParse(payloadOf(event))
      closed = true
      source.close()
      if (parsed.success) {
        handlersRef.current.onDone?.(parsed.data)
      }
    }

    const handleError = (event: Event) => {
      const parsed = indexErrorEventSchema.safeParse(payloadOf(event))
      closed = true
      source.close()
      if (parsed.success) {
        handlersRef.current.onError?.(parsed.data)
      }
    }

    source.addEventListener('progress', handleProgress)
    source.addEventListener('done', handleDone)
    source.addEventListener('error', handleError)

    source.onerror = () => {
      if (closed) {
        return
      }
      closed = true
      source.close()
      handlersRef.current.onDisconnect?.()
    }

    return () => {
      closed = true
      source.close()
      source.removeEventListener('progress', handleProgress)
      source.removeEventListener('done', handleDone)
      source.removeEventListener('error', handleError)
    }
  }, [enabled])
}

interface LiveProgress extends IndexProgressEvent {
  indexId: string
}

export interface BuildProgress {
  status: 'idle' | 'building' | 'ready' | 'error'
  progress: number
  processed: number | null
  total: number | null
  elapsedMs: number
  indexId: string
  error: string | null
  isLoading: boolean
  isError: boolean
  refetch: () => void
}

export function useBuildProgress(): BuildProgress {
  const statusQuery = useIndexStatus()
  const { refetch } = statusQuery
  const [live, setLive] = useState<LiveProgress | null>(null)
  const status = statusQuery.data
  const indexId = status?.indexId ?? ''

  const onProgress = useCallback(
    (event: IndexProgressEvent) => {
      setLive({ ...event, indexId })
    },
    [indexId],
  )

  const sync = useCallback(() => {
    setLive(null)
    void refetch()
  }, [refetch])

  useIndexEventStream(status?.status === 'building', {
    onProgress,
    onDone: sync,
    onError: sync,
    onDisconnect: () => setLive(null),
  })

  const active = live && live.indexId === indexId ? live : null

  return {
    status: status?.status ?? 'idle',
    progress: active?.progress ?? status?.progress ?? 0,
    processed: active?.processed ?? null,
    total: active?.total ?? null,
    elapsedMs: status?.elapsedMs ?? 0,
    indexId,
    error: status?.error ?? null,
    isLoading: statusQuery.isLoading,
    isError: statusQuery.isError,
    refetch: () => {
      void statusQuery.refetch()
    },
  }
}
