import { fileURLToPath, URL } from 'node:url'
import vue from '../../../../../../apps/web/node_modules/@vitejs/plugin-vue/dist/index.mjs'
import { defineConfig } from '../../../../../../apps/web/node_modules/vite/dist/node/index.js'

const local = (relative: string) => fileURLToPath(new URL(relative, import.meta.url))

// The fixture consumes local copies of the same versioned packages that
// external authors install from the registry.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: [
      { find: /^vue$/, replacement: local('../../../../../../apps/web/node_modules/vue/dist/vue.esm-bundler.js') },
      { find: /^@sforum\/admin-sdk$/, replacement: local('../../../../../../apps/web/packages/admin-sdk/src/index.ts') },
      { find: /^@sforum\/plugin-ui$/, replacement: local('../../../../../../apps/web/packages/plugin-ui/src/index.ts') },
      { find: /^@sforum\/plugin-ui\/style\.css$/, replacement: local('../../../../../../apps/web/packages/plugin-ui/src/style.css') }
    ]
  },
  build: {
    emptyOutDir: false,
    minify: 'esbuild',
    cssCodeSplit: false,
    rollupOptions: {
      input: local('./src/admin.ts'),
      preserveEntrySignatures: 'exports-only',
      output: {
        format: 'es',
        codeSplitting: false,
        entryFileNames: 'dashboard.mjs',
        assetFileNames: 'dashboard[extname]'
      }
    }
  }
})
