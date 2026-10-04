import { FastifyInstance } from 'fastify'

/** Reported by GET /api/version so clients can warn about a stale CLI. */
export interface VersionInfo {
    /** sculk-cli version this server was built against. */
    sculk: string
    /** Newest game version the bundled project scaffolding knows about. */
    gameVersion: string
    /** Data pack format for that game version. */
    packFormat: number
}

// Kept in step with src/commands/config (Sculk.Version) and
// src/commands/initProject/create (PackFormatFor). Override with
// SCULK_VERSION / SCULK_GAME_VERSION when releasing.
const versionInfo: VersionInfo = {
    sculk: process.env.SCULK_VERSION ?? '1.0.0',
    gameVersion: process.env.SCULK_GAME_VERSION ?? '26.2',
    packFormat: Number(process.env.SCULK_PACK_FORMAT ?? 107),
}

export default function versionRouteHandler(fastify: FastifyInstance) {
    fastify.get('/api/version', async (_request, reply) => {
        return reply.send(versionInfo)
    })
}
