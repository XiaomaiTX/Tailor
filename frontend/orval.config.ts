import { defineConfig } from 'orval'

export default defineConfig({
  starter: {
    input: { target: '../api/openapi/openapi.yaml' },
    output: {
      target: './src/api/generated/client.ts',
      schemas: './src/api/generated/model',
      client: 'react-query',
      mode: 'split',
      override: {
        mutator: { path: './src/api/http.ts', name: 'customFetch' },
      },
    },
  },
})
