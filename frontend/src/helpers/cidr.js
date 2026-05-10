// CIDR client-side validation helpers (BNet-m76e QoL).
//
// We only do shape checks here — final authority is the server-side
// validator on PUT /pools/{iface}, which enforces supernet containment +
// non-overlap with reserved/other-user pools and returns 400 with a
// descriptive error.

const V4_RE = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d|[12]\d|3[0-2])$/
const V6_RE = /^([0-9a-fA-F:]+)\/(\d|[1-9]\d|1[01]\d|12[0-8])$/

export function isValidCidrV4(s) {
  if (!s) return false
  const m = V4_RE.exec(s)
  if (!m) return false
  for (let i = 1; i <= 4; i++) {
    const o = parseInt(m[i], 10)
    if (o < 0 || o > 255) return false
  }
  return true
}

export function isValidCidrV6(s) {
  if (!s) return false
  if (!V6_RE.test(s)) return false
  // Loose v6 shape — full validation is server-side.
  const addr = s.split('/')[0]
  return addr.includes(':')
}

// Returns an error string for an invalid CIDR, or '' when OK / empty.
// `flavor` is one of 'v4' | 'v6' to pick the regex; v6 covers ULA + PI.
export function cidrError(s, flavor = 'v4') {
  if (!s) return '' // empty is OK (means "unset")
  if (flavor === 'v4') return isValidCidrV4(s) ? '' : 'invalid IPv4 CIDR'
  return isValidCidrV6(s) ? '' : 'invalid IPv6 CIDR'
}

// Returns true when child is contained in parent.
// Both must be valid CIDRs of the same family. Used for live "is this
// pool inside the supernet" check.
export function cidrContains(parent, child) {
  if (!parent || !child) return false
  if (parent === child) return true

  // Family must match.
  const v4Parent = isValidCidrV4(parent)
  const v4Child = isValidCidrV4(child)
  if (v4Parent !== v4Child) return false

  if (v4Parent) return v4Contains(parent, child)
  return v6Contains(parent, child)
}

function v4Contains(parent, child) {
  const [pAddr, pBitsStr] = parent.split('/')
  const [cAddr, cBitsStr] = child.split('/')
  const pBits = parseInt(pBitsStr, 10)
  const cBits = parseInt(cBitsStr, 10)
  if (cBits < pBits) return false // child must be more specific or equal
  const pNum = ipv4ToInt(pAddr) >>> (32 - pBits)
  const cNum = ipv4ToInt(cAddr) >>> (32 - pBits)
  return pNum === cNum
}

function ipv4ToInt(s) {
  const o = s.split('.').map(x => parseInt(x, 10))
  return ((o[0] << 24) | (o[1] << 16) | (o[2] << 8) | o[3]) >>> 0
}

function v6Contains(parent, child) {
  const [pAddr, pBitsStr] = parent.split('/')
  const [cAddr, cBitsStr] = child.split('/')
  const pBits = parseInt(pBitsStr, 10)
  const cBits = parseInt(cBitsStr, 10)
  if (cBits < pBits) return false

  const pBuf = expandV6(pAddr)
  const cBuf = expandV6(cAddr)
  if (!pBuf || !cBuf) return false

  let bitsLeft = pBits
  for (let i = 0; i < 16 && bitsLeft > 0; i++) {
    const take = Math.min(8, bitsLeft)
    const mask = take === 8 ? 0xff : (0xff << (8 - take)) & 0xff
    if ((pBuf[i] & mask) !== (cBuf[i] & mask)) return false
    bitsLeft -= take
  }
  return true
}

function expandV6(addr) {
  // Expand "fdcc::1" → "fdcc:0000:0000:0000:0000:0000:0000:0001" → byte buf.
  if (addr.includes('::')) {
    const [head, tail] = addr.split('::')
    const headParts = head ? head.split(':') : []
    const tailParts = tail ? tail.split(':') : []
    const fill = 8 - headParts.length - tailParts.length
    if (fill < 0) return null
    const parts = [...headParts, ...Array(fill).fill('0'), ...tailParts]
    return v6PartsToBytes(parts)
  }
  const parts = addr.split(':')
  if (parts.length !== 8) return null
  return v6PartsToBytes(parts)
}

function v6PartsToBytes(parts) {
  const bytes = new Uint8Array(16)
  for (let i = 0; i < 8; i++) {
    const v = parseInt(parts[i] || '0', 16)
    if (Number.isNaN(v)) return null
    bytes[i * 2] = (v >> 8) & 0xff
    bytes[i * 2 + 1] = v & 0xff
  }
  return bytes
}

// Returns true if cidrA and cidrB share any address.
export function cidrsOverlap(a, b) {
  return cidrContains(a, b) || cidrContains(b, a)
}
