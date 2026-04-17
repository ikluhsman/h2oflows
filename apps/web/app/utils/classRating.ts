// Converts numeric class ratings (e.g. 3.5) to Roman numeral display (e.g. "III+").

const ROMAN: Record<number, string> = {
  1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
  3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+',
  5: 'V', 5.5: 'V+', 6: 'VI',
}

export function romanClass(n: number): string {
  return ROMAN[n] ?? String(n)
}

export function classRange(min: number | null, max: number | null): string {
  if (min == null && max == null) return ''
  if (min == null) return romanClass(max!)
  if (max == null) return romanClass(min)
  if (min === max) return romanClass(min)
  return `${romanClass(min)}-${romanClass(max)}`
}
