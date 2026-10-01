import { FastifyInstance } from "fastify";

export default function versionRouteHandler(fastify: FastifyInstance) {
    fastify.get<Request, Response>()
}