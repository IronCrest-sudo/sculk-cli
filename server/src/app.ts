import fastify from 'fastify'
import 'dotenv/config'

const app = fastify()
const PORT = parseInt(process.env.PORT!) || 3000

// import route handlers


// listen
app.listen({port: PORT}, () => {
    console.log(`Listening on PORT ${PORT}! 🚀`)
})

