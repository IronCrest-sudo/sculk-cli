import fastify from 'fastify'
import 'dotenv/config'

import librariesRouteHandler from './routes/libraries'
import versionRouteHandler from './routes/version'

const app = fastify({
    logger: {
        level: process.env.LOG_LEVEL ?? 'info',
    },
})

// The website is usually served from a different origin than the API (Astro
// dev server vs. this one), so allow cross-origin reads of the listing.
const allowedOrigin = process.env.SCULK_CORS_ORIGIN ?? '*'
app.addHook('onRequest', async (_request, reply) => {
    reply.header('Access-Control-Allow-Origin', allowedOrigin)
    reply.header('Access-Control-Allow-Methods', 'GET,OPTIONS')
    reply.header('Access-Control-Allow-Headers', 'Accept, Content-Type')
    reply.header('Vary', 'Origin')
})

app.options('/*', async (_request, reply) => reply.status(204).send())

// import route handlers
app.register(librariesRouteHandler)
app.register(versionRouteHandler)

// health check
app.get('/api/health', async () => ({ ok: true }))

// JSON 404 instead of fastify's HTML-ish default
app.setNotFoundHandler((request, reply) => {
    reply.status(404).send({ error: `no route for ${request.method} ${request.url}` })
})

const PORT = parseInt(process.env.PORT ?? '3000', 10)
const HOST = process.env.HOST ?? '0.0.0.0'

// listen
app.listen({ port: PORT, host: HOST }, (err) => {
    if (err) {
        app.log.error(err)
        process.exit(1)
    }
    app.log.info(`Listening on ${HOST}:${PORT}! 🚀`)
})
