/**
 * featureIcons.ts
 *
 * Returns inline SVG HTML strings for feature type icons.
 * Used in the reach map sidebar and the features tabbed panel.
 * All content is static/trusted — safe to use with v-html.
 *
 * Icon style: 16×16 viewBox, white stroke/fill on a colored background circle.
 */

const S = 'stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"'

function svg(inner: string): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" style="display:block;width:100%;height:100%">${inner}</svg>`
}

/** Icon for access-point feature types (put_in, take_out, parking, camp, shuttle_drop, access). */
export function accessFeatureIcon(type: string): string {
  switch (type) {
    case 'put_in':
      // Arrow pointing down — into the water
      return svg(`<path ${S} d="M8 2v12M3 10l5 4 5-4"/>`)
    case 'take_out':
      // Arrow pointing up — out of the water
      return svg(`<path ${S} d="M8 14V2M3 6l5-4 5 4"/>`)
    case 'camp':
      // Tent triangle with internal ridge line
      return svg(`<path ${S} d="M2 14L8 2l6 12H2z"/><path ${S} d="M5.5 14L8 8l2.5 6"/>`)
    case 'parking':
      return svg(`<text fill="white" font-size="11" font-weight="900" font-family="system-ui,sans-serif" text-anchor="middle" dominant-baseline="middle" x="8" y="9">P</text>`)
    case 'shuttle_drop':
      return svg(`<text fill="white" font-size="11" font-weight="900" font-family="system-ui,sans-serif" text-anchor="middle" dominant-baseline="middle" x="8" y="9">S</text>`)
    default:
      // access / intermediate — solid diamond
      return svg(`<path fill="white" stroke="none" d="M8 3l3.5 5L8 13l-3.5-5z"/>`)
  }
}

/** Icon for rapids (regular or surf wave). */
export function rapidFeatureIcon(isSurf = false): string {
  if (isSurf) {
    // Cresting wave
    return svg(
      `<path stroke="white" stroke-width="2" stroke-linecap="round" fill="none" d="M1 5C3 2 5 2 7 5C9 8 11 8 13 5C14.5 3 15.5 2 15.5 2"/>` +
      `<path stroke="white" stroke-width="2" stroke-linecap="round" fill="none" d="M1 11C3 8 5 8 7 11C9 14 11 14 13 11"/>`
    )
  }
  // Two parallel wave lines
  return svg(
    `<path stroke="white" stroke-width="2" stroke-linecap="round" fill="none" d="M1 6C3 3 5 3 7 6C9 9 11 9 13 6"/>` +
    `<path stroke="white" stroke-width="2" stroke-linecap="round" fill="none" d="M1 11C3 8 5 8 7 11C9 14 11 14 13 11"/>`
  )
}

/** Icon for permanent hazards (low-head dam, strainer, etc.). */
export function hazardFeatureIcon(): string {
  return svg(
    `<path ${S} d="M8 2L1 14h14L8 2z"/>` +
    `<line stroke="white" stroke-width="2" stroke-linecap="round" x1="8" y1="7" x2="8" y2="11"/>` +
    `<circle fill="white" cx="8" cy="13" r="1"/>`
  )
}

/** Icon for flow gauge markers in the sidebar. */
export function gaugeFeatureIcon(relationship?: string | null): string {
  if (relationship === 'upstream_indicator') {
    // Chevron pointing up
    return svg(`<path ${S} d="M3 11l5-6 5 6"/>`)
  }
  if (relationship === 'downstream_indicator') {
    // Chevron pointing down
    return svg(`<path ${S} d="M3 5l5 6 5-6"/>`)
  }
  // Single wave line — generic gauge
  return svg(
    `<path stroke="white" stroke-width="2.5" stroke-linecap="round" fill="none" d="M1 8C3 5 5 5 7 8C9 11 11 11 13 8"/>`
  )
}

/**
 * Unified helper: returns the right icon for a feature given its type and flags.
 * Use for the features panel rows in slug.vue.
 */
export function featurePanelIcon(
  type: string,
  options: { isHazard?: boolean; isSurf?: boolean } = {}
): string {
  if (options.isHazard) return hazardFeatureIcon()
  if (type === 'rapid' || type === 'hazard') return rapidFeatureIcon(options.isSurf)
  return accessFeatureIcon(type)
}
