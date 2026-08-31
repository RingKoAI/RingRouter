// WebAuthn (passkey) browser helpers. The backend speaks the raw protocol
// payload with base64url-encoded binary fields, matching the JSON
// serialisation defined by the WebAuthn spec (go-webauthn v0.18 wire format):
//
//   finish bodies carry id / rawId / type / authenticatorAttachment /
//   clientExtensionResults / response{...}  — all optional envelope fields
//   are included so server-side parsing never depends on client quirks.

import { api, APIError } from './api'

export type PasskeyErrorCode =
  | 'cancelled'
  | 'disabled'
  | 'unsupported'
  | 'alreadyRegistered'
  | 'originMismatch'
  | 'sessionExpired'
  | 'rejected'

export class PasskeyError extends Error {
  code: PasskeyErrorCode

  constructor(code: PasskeyErrorCode, message: string = code) {
    super(message)
    this.name = 'PasskeyError'
    this.code = code
  }
}

export function passkeyErrorCode(error: unknown): PasskeyErrorCode {
  if (error instanceof PasskeyError) return error.code
  if (error instanceof APIError) {
    if (error.status === 503) return 'disabled'
    if (error.status === 401) return 'sessionExpired'
  }
  return 'rejected'
}

function browserPasskeyError(error: unknown): PasskeyError {
  if (error instanceof PasskeyError) return error
  if (error instanceof DOMException) {
    switch (error.name) {
      case 'NotAllowedError': return new PasskeyError('cancelled')
      case 'InvalidStateError': return new PasskeyError('alreadyRegistered')
      case 'SecurityError': return new PasskeyError('originMismatch')
      case 'NotSupportedError': return new PasskeyError('unsupported')
    }
  }
  if (error instanceof APIError) {
    if (error.status === 503) return new PasskeyError('disabled', error.message)
    if (error.status === 401) return new PasskeyError('sessionExpired', error.message)
    return new PasskeyError('rejected', error.message)
  }
  return new PasskeyError('rejected', error instanceof Error ? error.message : 'passkey rejected')
}

function b64uEncode(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf)
  let s = ''
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i])
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function b64uDecode(s: string): Uint8Array {
  const norm = s.replace(/-/g, '+').replace(/_/g, '/')
  const padded = norm + '='.repeat((4 - (norm.length % 4)) % 4)
  const raw = atob(padded)
  const out = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

// isPasskeySupported reports whether this browser/context can run WebAuthn.
// A non-secure context (plain HTTP on a non-localhost host) disables
// navigator.credentials entirely — surfacing that beats a raw TypeError.
export async function isPasskeySupported(): Promise<boolean> {
  if (typeof window === 'undefined' || !window.PublicKeyCredential) return false
  if (!window.isSecureContext) return false
  // A missing built-in platform authenticator does not mean WebAuthn is
  // unavailable: roaming USB/NFC keys and cross-device QR passkeys may still
  // work. Let navigator.credentials surface the actual device capability.
  return true
}

// Decodes the server options into types navigator.credentials understands.
function toPublicKeyCredentialOptions(options: any): PublicKeyCredentialCreationOptions {
  const pk = options.publicKey ?? options
  const out: any = {
    ...pk,
    challenge: b64uDecode(pk.challenge),
    user: pk.user ? { ...pk.user, id: b64uDecode(pk.user.id) } : undefined,
  }
  if (Array.isArray(pk.excludeCredentials)) {
    out.excludeCredentials = pk.excludeCredentials.map((c: any) => ({ ...c, id: b64uDecode(c.id) }))
  }
  // Empty attestationFormats arrays make some user agents reject the call.
  if (Array.isArray(out.attestationFormats) && out.attestationFormats.length === 0) {
    delete out.attestationFormats
  }
  return out as PublicKeyCredentialCreationOptions
}

function toAssertionOptions(options: any): PublicKeyCredentialRequestOptions {
  const pk = options.publicKey ?? options
  const out: any = {
    ...pk,
    challenge: b64uDecode(pk.challenge),
  }
  if (Array.isArray(pk.allowCredentials)) {
    out.allowCredentials = pk.allowCredentials.map((c: any) => ({ ...c, id: b64uDecode(c.id) }))
  }
  return out as PublicKeyCredentialRequestOptions
}

// Common envelope fields for finish payloads (go-webauthn CredentialCreationResponse /
// CredentialAssertionResponse both accept these at the top level).
function envelope(cred: PublicKeyCredential) {
  return {
    id: cred.id,
    rawId: b64uEncode(cred.rawId),
    type: cred.type,
    authenticatorAttachment: cred.authenticatorAttachment,
    clientExtensionResults: cred.getClientExtensionResults?.() ?? {},
  }
}

// Registers a new passkey for the signed-in user (requires a session cookie).
// Returns the server's passkey record on success.
export async function registerPasskey(name: string): Promise<void> {
  try {
    if (!(await isPasskeySupported())) throw new PasskeyError('unsupported')
    const begin = await api.post<{ challenge: string; options: any }>(
      '/api/auth/passkey/register/begin',
      {},
    )
    const opts = toPublicKeyCredentialOptions(begin.options)

    const cred = (await navigator.credentials.create({ publicKey: opts })) as PublicKeyCredential
    if (!cred) throw new PasskeyError('cancelled')

    const att = cred.response as AuthenticatorAttestationResponse
    const body = {
      ...envelope(cred),
      response: {
        clientDataJSON: b64uEncode(att.clientDataJSON),
        attestationObject: b64uEncode(att.attestationObject),
        transports: typeof att.getTransports === 'function' ? att.getTransports() : undefined,
      },
    }
    const res = await fetch(
      `/api/auth/passkey/register/finish?challenge=${encodeURIComponent(begin.challenge)}&name=${encodeURIComponent(name)}`,
      {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
      },
    )
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      if (res.status === 503) throw new PasskeyError('disabled', d.error)
      if (res.status === 401) throw new PasskeyError('sessionExpired', d.error)
      throw new PasskeyError('rejected', d.error || `HTTP ${res.status}`)
    }
  } catch (error) {
    throw browserPasskeyError(error)
  }
}

// Signs in with a passkey. username is optional: when empty the browser uses
// discoverable credentials. Returns true on success (session cookie set).
export async function loginWithPasskey(username?: string): Promise<boolean> {
  try {
    if (!(await isPasskeySupported())) throw new PasskeyError('unsupported')
    const begin = await api.post<{ challenge: string; options: any }>(
      '/api/auth/passkey/login/begin', username ? { username } : {},
    )
  // Unknown accounts receive a dead challenge (anti-enumeration); surface a
  // generic failure instead of leaking which names exist.
    if (!begin.options) throw new PasskeyError('rejected')

    const opts = toAssertionOptions(begin.options)
    const cred = (await navigator.credentials.get({ publicKey: opts })) as PublicKeyCredential
    if (!cred) throw new PasskeyError('cancelled')

    const resp = cred.response as AuthenticatorAssertionResponse
    const body = {
      ...envelope(cred),
      response: {
        clientDataJSON: b64uEncode(resp.clientDataJSON),
        authenticatorData: b64uEncode(resp.authenticatorData),
        signature: b64uEncode(resp.signature),
        userHandle: resp.userHandle ? b64uEncode(resp.userHandle) : null,
      },
    }
    const res = await fetch(`/api/auth/passkey/login/finish?challenge=${encodeURIComponent(begin.challenge)}`, {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    })
    if (!res.ok) {
      if (res.status === 503) throw new PasskeyError('disabled')
      if (res.status === 401) throw new PasskeyError('sessionExpired')
      throw new PasskeyError('rejected')
    }
    return true
  } catch (error) {
    throw browserPasskeyError(error)
  }
}
