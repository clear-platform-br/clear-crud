import { rm } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

const paths = [
  new URL('../packages/crud-client/dist/', import.meta.url),
  new URL('../packages/crud-vue/dist/', import.meta.url),
  new URL('../packages/crud-client/tsconfig.tsbuildinfo', import.meta.url),
  new URL('../packages/crud-vue/tsconfig.tsbuildinfo', import.meta.url),
]

await Promise.all(paths.map((url) => rm(fileURLToPath(url), { force: true, recursive: true })))
