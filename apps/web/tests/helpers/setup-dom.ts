// Bun test preload (see bunfig.toml [test].preload). Installs a deterministic
// happy-dom environment before any test file — and therefore before any `vue`,
// `@vue/test-utils`, or `@tiptap/vue-3` import — is evaluated. This removes the
// load-order race where a non-DOM file that imports Vue first freezes
// @vue/runtime-dom's internal `document` binding to `null`.
import { installTestDom } from './dom'

installTestDom()
