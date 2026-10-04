// The library listing.
//
// `libraries.json` next to this file is GENERATED from the CLI registry:
//
//     sculk list --json > website/src/data/libraries.json
//
// so the website can never drift from what `sculk add` actually accepts.
// Run `npm run gen:libraries` (see package.json) instead of editing the JSON
// by hand.
//
// The page renders from this file at build time and then, if a sculk API
// server is reachable on the same origin, refreshes itself from
// `/api/libraries` so a hosted listing can show newer entries without a
// rebuild. When no server is present the fetch fails silently and the static
// data stays on screen.

import staticEntries from './libraries.json';

export type PackKind = 'datapack' | 'resourcepack';

export interface LibraryEntry {
    identifier: string;
    description?: string;
    author?: string;
    source: string;
    subdir?: string;
    kind: PackKind;
    homepage?: string;
    notice?: string;
    installed?: string;
}

/** The build-time listing, exactly as the CLI emitted it. */
export const libraries: LibraryEntry[] = staticEntries as LibraryEntry[];

/**
 * Fetch a fresher listing from the sculk API server, if one is running.
 * Resolves to `null` when there is no server, it is unreachable, or it
 * answers with something unexpected - callers should keep the static data.
 */
export async function fetchLibraries(
    endpoint = '/api/libraries',
    timeoutMs = 2500,
): Promise<LibraryEntry[] | null> {
    try {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), timeoutMs);
        const res = await fetch(endpoint, {
            signal: controller.signal,
            headers: { Accept: 'application/json' },
        });
        clearTimeout(timer);
        if (!res.ok) return null;

        const data = await res.json();
        if (!Array.isArray(data)) return null;
        return data.filter(
            (e): e is LibraryEntry => !!e && typeof e.identifier === 'string',
        );
    } catch {
        return null;
    }
}

/** Group the listing by pack kind for display. */
export function byKind(entries: LibraryEntry[]): Record<string, LibraryEntry[]> {
    const out: Record<string, LibraryEntry[]> = {};
    for (const e of entries) {
        (out[e.kind] ??= []).push(e);
    }
    return out;
}
