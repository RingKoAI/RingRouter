// WebAuthn (passkey) browser helpers. The backend speaks the raw protocol
// payload with base64url-encoded binary fields, matching the JSON
// serialisation defined by the WebAuthn spec.

import { api } from './api'

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

// Decodes the server options into types navigator.credentials understands.
function toPublicKeyCredentialOptions(options: any): PublicKeyCredentialCreationOptions {
  const pk = options.publicKey ?? options
  return {
    ...pk,
    challenge: b64uDecode(pk.challenge),
    user: pk.user ? { ...pk.user, id: b64uDecode(pk.user.id) } : undefined,
    excludeCredentials: (pk.excludeCredentials ?? []).map((c: any) => ({
      ...c,
      id: b64uDecode(c.id),
    })),
  }
}

function toAssertionOptions(options: any): PublicKeyCredentialRequestOptions {
  const pk = options.publicKey ?? options
  return {
    ...pk,
    challenge: b64uDecode(pk.challenge),
    allowCredentials: (pk.allowCredentials ?? []).map((c: any) => ({
      ...c,
      id: b64uDecode(c.id),
    })),
  }
}

// Registers a new passkey for the signed-in user (requires a session cookie).
export async function registerPasskey(name: string): Promise<void> {
  const begin = await api.post<{ challenge: string; options: any }>(
    '/api/auth/passkey/register/begin',
    {},
  )
  const opts = toPublicKeyCredentialOptions(begin.options)

  const cred = (await navigator.credentials.create({ publicKey: opts })) as PublicKeyCredential
  if (!cred) throw new Error('passkey-cancelled')

  const body = {
    id: cred.id,
    rawId: b64uEncode(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: b64uEncode((cred.response as AuthenticatorAttestationResponse).clientDataJSON),
      attestationObject: b64uEncode((cred.response as AuthenticatorAttestationResponse).attestationObject),
      transports: (cred.response as AuthenticatorAttestationResponse).getTransports?.() ?? [],
    },
  }
  await fetch(
    `/api/auth/passkey/register/finish?challenge=${encodeURIComponent(begin.challenge)}&name=${encodeURIComponent(name)}`,
    {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  ).then(async (res) => {
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error(d.error || `HTTP ${res.status}`)
    }
  })
}

// Signs in with a passkey. username is optional: when empty the browser uses
// discoverable credentials. Returns true on success (session cookie set).
export async function loginWithPasskey(username?: string): Promise<boolean> {
  const begin = await api.post<{ challenge: string; options: any }>(
    '/api/auth/passkey/login/begin',
    username ? { username } : {},
  )
  // Unknown accounts receive a dead challenge (anti-enumeration); surface a
  // generic failure instead of leaking which names exist.
  if (!begin.options) throw new Error('passkey-failed')

  const opts = toAssertionOptions(begin.options)
  const cred = (await navigator.credentials.get({ publicKey: opts })) as PublicKeyCredential
  if (!cred) throw new Error('passkey-cancelled')

  const resp = cred.response as AuthenticatorAssertionResponse
  const body = {
    id: cred.id,
    rawId: b64uEncode(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: b64uEncode(resp.clientDataJSON),
      authenticatorData: b64uEncode(resp.authenticatorData),
      signature: b64uEncode(resp.signature),
      userHandle: resp.userHandle ? b64uEncode(resp.userHandle) : undefined,
    },
  }
  const res = await fetch(
    `/api/auth/passkey/login/finish?challenge=${encodeURIComponent(begin.challenge)}`,
    {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  if (!res.ok) throw new Error('passkey-failed')
  return true
}
