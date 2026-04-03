const baseUrl = process.env.API_BASE_URL || 'http://localhost:3000/api'

async function check(path) {
  const response = await fetch(`${baseUrl}${path}`)
  const payload = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw new Error(`${path} failed: ${payload.message || response.status}`)
  }

  console.log(`ok ${path}`)
}

async function run() {
  await check('/health')
  await check('/dashboard/summary')
  await check('/graph/subgraph')
  process.exit(0)
}

run().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
