class StubEventSource extends EventTarget {
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSED = 2
  readonly CONNECTING = 0
  readonly OPEN = 1
  readonly CLOSED = 2
  readonly url: string
  readonly withCredentials = false
  readyState = 0
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onopen: ((event: Event) => void) | null = null

  constructor(url: string | URL) {
    super()
    this.url = String(url)
  }

  close() {
    this.readyState = 2
  }
}

if (typeof EventSource === 'undefined') {
  globalThis.EventSource = StubEventSource as unknown as typeof EventSource
}
