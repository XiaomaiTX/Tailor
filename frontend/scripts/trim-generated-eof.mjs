import { readFile, writeFile } from 'node:fs/promises'

const target = new URL('../src/api/generated/client.ts', import.meta.url)
const source = await readFile(target, 'utf8')
const normalized = source.replace(/(?:\r?\n){2,}$/, '\n')

if (source !== normalized) {
  await writeFile(target, normalized, 'utf8')
}
