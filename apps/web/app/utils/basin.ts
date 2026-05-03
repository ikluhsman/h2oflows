export function cleanBasinName(name: string | null): string | null {
  if (!name) return null
  const cleaned = name
    .replace(/^(Upper|Middle|Lower)\s+/i, '')
    .replace(/\s+(River|Rivers|Basin)s?$/i, '')
    .trim()
  return cleaned || null
}

export function slugifyBasin(name: string): string {
  return (cleanBasinName(name) ?? name)
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}
