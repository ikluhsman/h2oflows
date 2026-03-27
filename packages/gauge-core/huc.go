package gauge

// HUCNames derives a human-readable basin name and watershed name from a
// USGS 8-digit Hydrologic Unit Code (HUC8).
//
// The HUC hierarchy:
//   HUC2  (2 digits) — major drainage region   e.g. "14" = Upper Colorado
//   HUC4  (4 digits) — subregion / main stem    e.g. "1402" = Gunnison River
//   HUC6  (6 digits) — accounting unit
//   HUC8  (8 digits) — cataloging unit / watershed
//
// We derive:
//   basin_name     — the ultimate major drainage basin (HUC2)
//   watershed_name — the main stem river system (HUC4), used for UI grouping
//
// This handles disambiguation automatically:
//   Escalante Creek, CO → HUC4 1402 → "Gunnison River" watershed, "Colorado River Basin"
//   Escalante River, UT → HUC4 1406 → "Lower Colorado / Escalante" watershed, "Colorado River Basin"
//
// Returns ("", "") if the code is unrecognized — callers should treat this as
// "populate from name heuristics or leave for manual entry."
func HUCNames(huc8 string) (basinName, watershedName string) {
	if len(huc8) < 4 {
		return "", ""
	}
	huc2 := huc8[:2]
	huc4 := huc8[:4]

	basinName = huc2Basin(huc2)
	watershedName = huc4Watershed(huc4)
	return
}

// huc2Basin maps the 2-digit HUC region code to a major drainage basin name.
func huc2Basin(huc2 string) string {
	switch huc2 {
	case "10":
		return "Missouri River Basin"
	case "11":
		return "Arkansas River Basin"
	case "12":
		return "Texas Gulf Basin"
	case "13":
		return "Rio Grande Basin"
	case "14":
		return "Colorado River Basin"
	case "15":
		return "Lower Colorado Basin"
	case "16":
		return "Great Basin"
	case "17":
		return "Pacific Northwest Basin"
	case "18":
		return "California Basin"
	default:
		return ""
	}
}

// huc4Watershed maps the 4-digit HUC subregion to the main stem river system.
// These are the named river systems users and paddlers recognize.
func huc4Watershed(huc4 string) string {
	switch huc4 {
	// ----- Colorado River Basin (HUC2 = 14) -----------------------------------
	case "1401":
		return "Upper Colorado River"
	case "1402":
		return "Gunnison River"
	case "1403":
		return "White-Yampa Rivers"
	case "1404":
		return "Lower Green River"
	case "1405":
		return "Upper Green River"
	case "1406":
		return "Lower Colorado / Escalante" // Utah — includes Escalante River UT
	case "1407":
		return "Glen Canyon / Colorado"
	case "1408":
		return "Little Colorado River"

	// ----- Lower Colorado Basin (HUC2 = 15) ----------------------------------
	case "1501":
		return "Lower Colorado River"
	case "1502":
		return "Bill Williams River"
	case "1503":
		return "Sonoran Desert Rivers"

	// ----- Arkansas–White–Red Basin (HUC2 = 11) ------------------------------
	case "1101":
		return "Upper Arkansas River"
	case "1102":
		return "Middle Arkansas River"
	case "1103":
		return "Lower Arkansas River"
	case "1110":
		return "Upper Arkansas River" // Upper Ark headwaters (alt subregion)

	// ----- Rio Grande Basin (HUC2 = 13) --------------------------------------
	case "1301":
		return "Upper Rio Grande"
	case "1302":
		return "Middle Rio Grande"
	case "1303":
		return "Lower Rio Grande"
	case "1304":
		return "Pecos River"
	case "1306":
		return "Closed Basins / San Luis Valley"
	case "1308":
		return "Upper Rio Grande"

	// ----- Missouri River Basin (HUC2 = 10) ----------------------------------
	// South Platte and North Platte are in the Missouri system
	case "1018":
		return "South Platte River"
	case "1019":
		return "North Platte River"
	case "1023":
		return "North Platte River"
	case "1024":
		return "Upper Missouri River"
	case "1025":
		return "Yellowstone River"
	case "1026":
		return "Middle Missouri River"

	// ----- Great Basin (HUC2 = 16) -------------------------------------------
	case "1601":
		return "Bear River"
	case "1602":
		return "Great Salt Lake"
	case "1603":
		return "Sevier River"

	// ----- Pacific Northwest (HUC2 = 17) -------------------------------------
	case "1701":
		return "Columbia River Headwaters"
	case "1702":
		return "Snake River"

	default:
		return ""
	}
}
