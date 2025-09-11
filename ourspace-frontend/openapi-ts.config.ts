import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../ourspace-backend/openapi/openapi.yaml',
  output: 'src/client',
  plugins: ['@hey-api/client-fetch'],
});
