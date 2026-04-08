# KMZ Import Guide

How to structure a Google My Map so its KMZ export imports cleanly into H2OFlows.

The importer reads pin name prefixes (`Rapid:`, `Put-in:`, etc.) and folder names to map placemarks to rapids, put-ins, take-outs, parking spots, and shuttle drops. Following the conventions below means you can build a reach in Google My Maps and import it with one command — no manual SQL.

> Reference implementation: [apps/api/internal/kmlimport/kmlimport.go](../apps/api/internal/kmlimport/kmlimport.go)

---

## Two import modes

The importer auto-detects which mode your map uses.

### Mode A — Folder-per-reach *(recommended for single-river maps)*

Each top-level folder = one reach. The folder name must match a reach's `name` or `slug` (case-insensitive, partial match works).

```
My Map
├── Browns Canyon                ← matches reach by name
│   ├── Rapid: Zoom Flume
│   ├── Rapid: Big Drop
│   ├── Put-in: Fisherman's Bridge
│   ├── Take-out: Hecla Junction
│   └── Parking: Hecla Lot
└── Royal Gorge
    ├── Put-in: Parkdale
    ├── Take-out: Cañon City
    └── Rapid: Sunshine Falls
```

The importer matches the folder name against the `reaches` table using:

1. Exact name match
2. Exact slug match
3. Substring match (folder contains reach name, or vice versa)

If no reach matches, the entire folder is skipped with a warning.

### Mode B — Category-organized *(for regional maps spanning many reaches)*

Folders are named by feature type, and the importer infers which reach each pin belongs to. All folder names must come from this set:

- `Access Points` / `Access`
- `Rivers` / `Waterways` / `River Lines`
- `Rapids`
- `Features`

```
Colorado Whitewater
├── Access Points
│   ├── Browns Canyon — Fisherman's Bridge put-in
│   ├── Browns Canyon — Hecla Junction take-out
│   └── Numbers — Granite Bridge put-in
├── Rapids
│   ├── Numbers — Number 5
│   └── Browns — Zoom Flume
└── Rivers
    └── (line strings — currently ignored)
```

In category mode, reach assignment happens in two passes:

1. **Name-based:** the importer searches each pin's `name + description` for any keyword from a reach name (excluding generic words like "river", "creek", "canyon", "fork", "upper", "lower"). First match wins.
2. **Proximity fallback:** any pin that didn't name-match gets assigned to the geographically nearest pin that *did* name-match.

This means **at least one pin per reach must mention the reach name** to anchor the others by proximity.

---

## Pin naming conventions

The importer reads a prefix off each pin name to decide what kind of feature it is.

| Prefix | Stored as | Example |
|---|---|---|
| `Rapid:` | `rapids` row | `Rapid: Zoom Flume` |
| `Wave:` / `Surf:` | `rapids` row, `is_surf_wave = true` | `Surf: Glenwood Wave` |
| `Put-in:` | `reach_access` type=`put_in` | `Put-in: Fisherman's Bridge` |
| `Take-out:` | `reach_access` type=`take_out` | `Take-out: Hecla Junction` |
| `Parking:` | `reach_access` type=`parking` | `Parking: Hecla Lot` |
| `Shuttle:` | `reach_access` type=`shuttle_drop` | `Shuttle: Buena Vista` |

The colon is required. The text after the colon becomes the feature name.

### Description-based fallback

If you forget a prefix, the importer tries to infer the feature type from the description text. Keywords it looks for:

- `parking`, `can park`, `park here` → `parking`
- `take-out`, `takeout`, `take out` → `take-out`
- `put-in`, `put in` → `put-in`
- `surf wave`, `surf spot`, `surfable`, `play wave` → `wave`
- `class`, `line is`, `boof`, `ledge` → `rapid`

And as a last resort, the folder name (in category mode) is used as a hint:

- `Rapids` / `Waves` → `rapid`
- `Access Points` / `Access` → `put-in`

This is fragile — **prefer the explicit prefix.**

### Class ratings

If you put `Class III+` (or `Class V`, `Class IV-`, etc.) anywhere in a rapid's description, the importer extracts it into `rapids.class_rating` as a float (`3.5`, `5.0`, `3.75`).

```
Rapid: Zoom Flume
Description: Class IV. Big wave train, river-right line cleanest.
```

---

## What gets replaced on re-import

For each reach the importer touches, **rapids and access points with `data_source = 'import'` or `'ai_seed'` are deleted first**, then re-inserted from the KMZ.

This means you can re-export your Google My Map and re-import safely — your latest version replaces the previous one. Rapids and accesses created with `data_source = 'maintainer'` (manually authored, not from a KMZ or AI seed) are never deleted.

Geometry-only updates are not supported — you must re-import the full reach.

---

## Importing

```bash
cd apps/api
/usr/local/go/bin/go run ./cmd/import-kml -file /path/to/your-export.kmz
```

Add `-dry-run` to see what would be imported without touching the database.

The importer prints a per-reach summary and a log of every pin it processed:

```
Browns Canyon: 12 rapids, 1 put-in, 1 take-out, 2 parking
Royal Gorge:    8 rapids, 1 put-in, 1 take-out

✓ [Browns Canyon] rapid: Zoom Flume
✓ [Browns Canyon] put-in: Fisherman's Bridge
~ "Hecla Lot" → Browns Canyon (by proximity)
⚠  folder "Numbers" — no matching reach, skipping
```

Symbols:

- `✓` — pin imported successfully
- `~` — pin assigned to a reach by proximity (not by name)
- `↺` — previous import data cleared for this reach
- `⚠` — warning, pin or folder skipped
- `✗` — error during insert

---

## Tips for clean Google My Maps

- **One reach per folder** is the easiest mode. Use it unless you really need a regional overview map.
- **Prefix every pin.** Don't rely on description-based inference.
- **Anchor at least one pin per reach with the reach name** in category mode, so proximity matching has something to grab onto.
- **Don't use generic words alone** as folder names — `River` or `Creek` won't match anything.
- **Line strings (river centerlines) are ignored** by the importer. Reach geometry comes from OSM via the centerline fetcher, not KMZ.
- **Re-export and re-import freely.** It's idempotent for `import`-sourced data.

---

## Troubleshooting

**"folder X — no matching reach, skipping"**
The folder name doesn't match any reach. Check spelling, or rename the folder to match the reach's slug exactly.

**"Y — no anchors, skipping"** *(category mode)*
No pin in the map name-matched a reach, so proximity fallback has nothing to work from. Add a pin whose name explicitly contains the reach name.

**"unknown type, skipping"**
The pin had no prefix and the description didn't match any inference keyword. Add an explicit `Rapid:` / `Put-in:` / etc. prefix.

**Pins disappeared after re-import**
Expected — `import`-sourced rows are cleared before each re-import. If you want a pin to survive, set its `data_source = 'maintainer'` manually in the database.
