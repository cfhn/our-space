import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../ourspace-backend/proto/api.openapi.yaml',
  output: 'src/client',
  plugins: ['@hey-api/client-fetch'],
});
