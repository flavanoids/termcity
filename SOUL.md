# TermCity — Soul Document

## What This Project Is

TermCity is a tool for people who live near emergency activity and want to understand what's happening around them — without opening a browser, without ads, without requiring a smartphone. It runs in a terminal because terminals are universal, lightweight, and present on every server, embedded system, and developer machine on earth.

It is not a dispatch tool. It is not a replacement for official emergency information. It is a window — a way to satisfy the human instinct to understand one's environment.

## Core Beliefs

### The Terminal Is Sufficient
Every feature in TermCity asks: can this be done well in a terminal? The answer is almost always yes. Half-block pixel rendering isn't a compromise — it's a deliberate choice to do something visually rich within constraints. Those constraints produce character.

### Respect the Data Sources
OSM, Nominatim, PulsePoint, and city Socrata portals are public goods operated by volunteers and governments. TermCity is a guest on their infrastructure. Rate limits are not obstacles — they are agreements. The 1 req/s Nominatim limit isn't a bug to work around; it's a responsibility to honor.

### Fail Gracefully, Not Loudly
Emergency data is inconsistent. APIs go down. Cities change their Socrata field names. The PulsePoint API format shifts without notice. TermCity should handle all of this without crashing, without showing the user a stack trace, and without pretending everything is fine when it isn't. A warning in the sidebar is better than a panic.

### Simplicity Over Completeness
TermCity doesn't try to support every city's data source. It supports the ones it can support well. A partial city registry that works reliably is better than a complete one that fails constantly.

### Privacy by Default
TermCity makes no network requests except:
1. Tile fetches from OSM (no user data sent)
2. Geocoding from Nominatim (only the zip code)
3. Incident data from PulsePoint/Socrata (no user data sent)

There is no telemetry. There is no analytics. There is no account system. The zip code never leaves the machine except to resolve it to coordinates.

## What TermCity Should Feel Like

When you open TermCity and enter your zip code, it should feel like looking at a real map. The tiles should load quickly (cache helps). The incidents should pulse gently — noticeable but not alarming. The sidebar should be scannable in two seconds.

When you press `?`, the help overlay should feel like a cheat sheet, not a manual. When you press `q`, it should exit immediately.

The overall feel should be: **calm competence**. Not exciting. Not alarming. Just a clear, accurate picture of what's happening nearby.

## What TermCity Should Never Become

- **A news ticker**: Incident titles should be factual call types, not sensationalized descriptions.
- **An alert system**: TermCity does not notify you of anything. It shows you what's already been dispatched.
- **A surveillance tool**: Incidents are shown as dots on a map. No personal information about callers or responders is displayed.
- **A resource hog**: The app should use minimal CPU when idle (the 300ms animation tick is the primary loop). Tile fetches are rate-limited for a reason.

## The Right Amount of Features

The current feature set is intentional:
- Zip code → geocoded center
- OSM tiles at multiple zoom levels
- PulsePoint fire/EMS incidents
- Socrata police incidents for supported cities
- Pulsing incident markers
- Scrollable sidebar
- Help overlay

Features that might seem obvious but are deliberately excluded:
- **Sound alerts**: This is a display tool, not a pager.
- **Multiple zip codes**: Focus on one area at a time.
- **Incident history**: Show what's active now, not what happened last week.
- **User accounts / saved locations**: Unnecessary complexity.

If a feature request doesn't serve the core purpose — showing what's happening nearby right now — it probably doesn't belong here.

## On the Use of OSM

OpenStreetMap is a community project built by volunteers. Every tile served is paid for by OSM's community. TermCity uses OSM tiles because they are the best freely available map tiles, and because the OSM community has explicitly allowed this use with proper attribution and rate limiting.

If TermCity ever becomes popular enough to strain OSM's tile servers, the right response is to implement a tile server cache proxy, not to remove rate limits.

## The Name

TermCity: **Term** for terminal, **City** for the urban environment where 911 activity is most visible. It's a simple name for a simple idea.
