import { FastifyInstance } from 'fastify'
import { readFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

/**
 * One entry of the sculk library registry, as emitted by `sculk list --json`.
 * Kept in sync by hand with src/commands/list/main.go (Entry).
 */
export interface LibraryEntry {
    identifier: string
    description?: string
    author?: string
    source: string
    subdir?: string
    kind: 'datapack' | 'resourcepack'
    homepage?: string
    notice?: string
    installed?: string
}

const here = path.dirname(fileURLToPath(import.meta.url))

/**
 * Where the registry JSON lives.
 *
 * The generated file is shared with the website so that the CLI, the site and
 * the API can never disagree. Override with SCULK_LIBRARIES_JSON when the
 * server is deployed away from the monorepo checkout.
 */
function registryPath(): string {
    return (
        process.env.SCULK_LIBRARIES_JSON ??
        path.resolve(here, '../../../website/src/data/libraries.json')
    )
}

async function loadRegistry(): Promise<LibraryEntry[]> {
    const file = registryPath()
    if (!existsSync(file)) {
        throw Object.assign(
            new Error(
                `registry file not found at ${file}. Generate it with ` +
                `'sculk list --json > website/src/data/libraries.json', ` +
                'or point SCULK_LIBRARIES_JSON at it.',
            ),
            { statusCode: 500 },
        )
    }
    const raw = await readFile(file, 'utf8')
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) {
        throw Object.assign(
            new Error(`registry file ${file} does not contain a JSON array`),
            { statusCode: 500 },
        )
    }
    return parsed as LibraryEntry[]
}

export default function librariesRouteHandler(fastify: FastifyInstance) {
    // GET /api/libraries - the whole registry, optionally narrowed with ?q=
    fastify.get('/api/libraries', async (request, reply) => {
        let entries: LibraryEntry[]
        try {
            entries = await loadRegistry()
        } catch (err) {
            request.log.error(err)
            return reply
                .status((err as { statusCode?: number }).statusCode ?? 500)
                .send({ error: (err as Error).message })
        }

        const { q, kind } = request.query as { q?: string; kind?: string }

        if (kind) {
            entries = entries.filter((e) => e.kind === kind)
        }

        if (q) {
            const needle = q.trim().toLowerCase()
            if (needle !== '') {
                entries = entries.filter((e) =>
                    [e.identifier, e.description, e.author, e.source, e.subdir, e.kind]
                        .filter((v): v is string => typeof v === 'string')
                        .some((v) => v.toLowerCase().includes(needle)),
                )
            }
        }

        return reply.send(entries)
    })

    // GET /api/libraries/:identifier - one entry, or 404
    fastify.get('/api/libraries/:identifier', async (request, reply) => {
        const { identifier } = request.params as { identifier: string }

        let entries: LibraryEntry[]
        try {
            entries = await loadRegistry()
        } catch (err) {
            request.log.error(err)
            return reply
                .status((err as { statusCode?: number }).statusCode ?? 500)
                .send({ error: (err as Error).message })
        }

        const found = entries.find(
            (e) => e.identifier.toLowerCase() === identifier.toLowerCase(),
        )
        if (!found) {
            return reply.status(404).send({ error: `unknown library '${identifier}'` })
        }
        return reply.send(found)
    })
}
