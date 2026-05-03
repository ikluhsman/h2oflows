// Converts numeric class ratings (e.g. 3.5) to Roman numeral display (e.g. "III+").

const ROMAN: Record<number, string> = {
  1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
  3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+',
  5: 'V', 5.5: 'V+', 6: 'VI',
}

export function romanClass(n: number): string {
  return ROMAN[n] ?? String(n)
}

// Matches the map's classColorExpr step expression.
export function classColor(n: number): string {
  if (n < 0.5) return '#6b7280'   // gray  — no class
  if (n < 3.0) return '#16a34a'   // green — I–II
  if (n < 4.0) return '#3b82f6'   // blue  — III
  if (n < 5.0) return '#111827'   // black — IV
  return '#dc2626'                  // red   — V+
}

export function classRange(min: number | null, max: number | null): string {
  if (min == null && max == null) return ''
  if (min == null) return romanClass(max!)
  if (max == null) return romanClass(min)
  if (min === max) return romanClass(min)
  return `${romanClass(min)}-${romanClass(max)}`
}
