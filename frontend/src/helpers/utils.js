import { Address4, Address6 } from "ip-address"

export function ipToBigInt(ip) {
  // Check if it's an IPv4 address
  if (ip.includes(".")) {
    const addr = new Address4(ip)
    return addr.bigInt()
  }

  // Otherwise, assume it's an IPv6 address
  const addr = new Address6(ip)
  return addr.bigInt()
}

export function humanFileSize(size) {
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"]
  if (size === 0) return "0B"
  const i = parseInt(Math.floor(Math.log(size) / Math.log(1024)))
  return Math.round(size / Math.pow(1024, i), 2) + sizes[i]
}

// humanRelativeTime returns a "Created N units ago" style string for an
// ISO-8601 timestamp. Used by both PeerEditModal and the peer list table
// so both surfaces share the same age formatting.
//   - 'just now' (< 1 minute, or future timestamps)
//   - 'N minute(s) ago' / 'N hour(s) ago' / 'N day(s) ago'
//   - 'on YYYY-MM-DD' (>= 30 days)
// Returns empty string when input is falsy or unparseable. Caller passes
// in vue-i18n's t() so this stays language-agnostic.
export function humanRelativeTime(iso, t) {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const ms = Date.now() - d.getTime()
  if (ms < 60000) return t('general.time.just-now')
  const minutes = Math.floor(ms / 60000)
  const hours = Math.floor(ms / 3600000)
  const days = Math.floor(ms / 86400000)
  if (days >= 30) return t('general.time.on-date', { date: d.toLocaleDateString() })
  if (days >= 1) return t('general.time.days-ago', { n: days })
  if (hours >= 1) return t('general.time.hours-ago', { n: hours })
  return t('general.time.minutes-ago', { n: minutes })
}
