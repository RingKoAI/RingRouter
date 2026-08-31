const BASE_URL = import.meta.env.VITE_API_URL || ''

export class APIError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

// Session-expiry hook: when any request comes back 401 the management
// session is gone — bounce to the login page once instead of letting every
// page silently degrade into empty states.
let unauthorizedHandler: (() => void) | null = null

export function onUnauthorized(handler: (() => void) | null) {
  unauthorizedHandler = handler
}

let lastUnauthorizedAt = 0

function handleUnauthorized() {
  if (!unauthorizedHandler) return
  // Debounce: concurrent failed calls must not fan out into repeated calls.
  const now = Date.now()
  if (now - lastUnauthorizedAt < 2000) return
  lastUnauthorizedAt = now
  unauthorizedHandler()
}

async function request<T = any>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    credentials: 'same-origin',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  let data: any = null
  try {
    data = await res.json()
  } catch {
    // Non-JSON response body (proxy error, empty body, etc.)
  }

  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/api/auth/')) {
      handleUnauthorized()
    }
    throw new APIError(res.status, data?.error || `HTTP ${res.status}`)
  }

  return data as T
}

export const api = {
  get: <T = any>(path: string) => request<T>('GET', path),
  post: <T = any>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T = any>(path: string, body: unknown) => request<T>('PUT', path, body),
  delete: <T = any>(path: string) => request<T>('DELETE', path),
}
