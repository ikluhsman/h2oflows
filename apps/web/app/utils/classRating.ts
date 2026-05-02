// Converts numeric class ratings (e.g. 3.5) to Roman numeral display (e.g. "III+").

const ROMAN: Record<number, string> = {
  1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
  3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+',
  5: 'V', 5.5: 'V+', 6: 'VI',
}

export function romanClass(n: number): string {
  return ROMAN[n] ?? String(n)
}

export function classColor(n: number): string {
  if (n <= 1.5) return '#22c55e'
  if (n <= 2.5) return '#84cc16'
  if (n <= 3.5) return '#eab308'
  if (n <= 4.5) return '#f97316'
  if (n <= 5.5) return '#ef4444'
  return '#7f1d1d'
}

export function classRange(min: number | null, max: number | null): string {
  if (min == null && max == null) return ''
  if (min == null) return romanClass(max!)
  if (max == null) return romanClass(min)
  if (min === max) return romanClass(min)
  return `${romanClass(min)}-${romanClass(max)}`
}
