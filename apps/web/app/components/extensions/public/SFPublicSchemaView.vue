<script setup lang="ts">
import { computed } from 'vue'

/**
 * `view: schema` 公开页的正文渲染器。
 *
 * 契约（决策记录 2026-10-16 冻结）：页面只声明一个**有界**页面描述符
 * （list | detail | form | notice），复用 AdminSurface 的视图模型/字段形状，
 * 不含 Vue、不含 Go 模板、无需构建步骤。宿主负责渲染。
 *
 * 与 admin 的 surface outlet 刻意分开：管理端 outlet 带可调用命令、资源可见性与
 * 下载等语义，公开页不应继承这些能力。这里只渲染数据本身。
 */
type SchemaOption = { label?: string, value?: unknown }
type SchemaField = {
  key?: string
  label?: string
  type?: string
  required?: boolean
  placeholder?: string
  options?: SchemaOption[]
}

const props = defineProps<{
  descriptor: Record<string, unknown>
  extensionId: string
}>()

const kind = computed(() => String(props.descriptor?.kind ?? ''))
const title = computed(() => text(props.descriptor?.title))
const description = computed(() => text(props.descriptor?.description))
const message = computed(() => text(props.descriptor?.message))
const tone = computed(() => text(props.descriptor?.tone) || 'neutral')
const items = computed(() => arrayOf(props.descriptor?.items))
const cells = computed(() => {
  const raw = props.descriptor?.cells
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return []
  return Object.entries(raw as Record<string, unknown>).map(([key, value]) => ({ key, value: text(value) }))
})
const fields = computed<SchemaField[]>(() =>
  arrayOf(props.descriptor?.fields).map(field => ({
    key: text(field.key),
    label: text(field.label),
    type: text(field.type) || 'text',
    required: field.required === true,
    placeholder: text(field.placeholder),
    options: arrayOf(field.options)
  }))
)

function text(value: unknown): string {
  if (value === null || value === undefined) return ''
  return typeof value === 'string' ? value : String(value)
}

function arrayOf(value: unknown): any[] {
  return Array.isArray(value) ? value : []
}

function fieldValue(field: SchemaField): string {
  const raw = (props.descriptor?.values as Record<string, unknown> | undefined)?.[field.key ?? '']
  return text(raw)
}

function isChecked(field: SchemaField): boolean {
  return (props.descriptor?.values as Record<string, unknown> | undefined)?.[field.key ?? ''] === true
}
</script>

<template>
  <section
    class="sf-public-schema-view"
    :data-schema-kind="kind"
    :data-schema-tone="tone"
    :data-schema-extension="extensionId"
  >
    <header v-if="title || description" class="mb-3">
      <h2 v-if="title" class="text-xl font-semibold" data-schema-title>
        {{ title }}
      </h2>
      <p v-if="description" class="mt-1 text-sm text-muted" data-schema-description>
        {{ description }}
      </p>
    </header>

    <!-- list：条目 + 单元格表格 -->
    <div v-if="kind === 'list'" class="space-y-3">
      <ul v-if="items.length" class="space-y-1" data-schema-items>
        <li v-for="(item, index) in items" :key="index" class="flex items-baseline gap-2">
          <span data-schema-item-label>{{ text(item?.label) }}</span>
          <span class="text-muted" data-schema-item-value>{{ text(item?.value) }}</span>
        </li>
      </ul>
      <table v-if="cells.length" class="w-full text-sm" data-schema-cells>
        <tbody>
          <tr v-for="cell in cells" :key="cell.key">
            <th class="pr-3 text-left font-medium">{{ cell.key }}</th>
            <td>{{ cell.value }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- detail：条目按 label/value 成对展示 -->
    <dl v-else-if="kind === 'detail'" class="space-y-1" data-schema-items>
      <div v-for="(item, index) in items" :key="index" class="flex gap-2">
        <dt class="font-medium">{{ text(item?.label) }}</dt>
        <dd class="text-muted">{{ text(item?.value) }}</dd>
      </div>
    </dl>

    <!-- notice：一条消息 -->
    <div v-else-if="kind === 'notice'" class="rounded-md border p-3" data-schema-notice>
      <p>{{ message || description }}</p>
    </div>

    <!--
      form：字段只读呈现。提交需要页面声明的 dataRoute 绑定，而插件页数据装载
      仍受 loader 传输缺口阻塞（与 dataSource=plugin 同一阻塞项），因此这里不伪造提交。
    -->
    <div v-else-if="kind === 'form'" class="space-y-3" data-schema-form-pending="1">
      <div v-for="field in fields" :key="field.key" class="space-y-1">
        <label :for="`sf-schema-${field.key}`" class="block text-sm font-medium">
          {{ field.label }}
          <span v-if="field.required" class="text-muted">*</span>
        </label>
        <select
          v-if="field.type === 'select'"
          :id="`sf-schema-${field.key}`"
          class="w-full"
          disabled
        >
          <option
            v-for="option in field.options || []"
            :key="text(option.value)"
            :value="text(option.value)"
            :selected="text(option.value) === fieldValue(field)"
          >
            {{ text(option.label) }}
          </option>
        </select>
        <input
          v-else-if="field.type === 'boolean'"
          :id="`sf-schema-${field.key}`"
          type="checkbox"
          :checked="isChecked(field)"
          disabled
        >
        <textarea
          v-else-if="field.type === 'textarea'"
          :id="`sf-schema-${field.key}`"
          class="w-full"
          :placeholder="field.placeholder"
          disabled
        />
        <input
          v-else
          :id="`sf-schema-${field.key}`"
          :type="field.type === 'number' ? 'number' : 'text'"
          class="w-full"
          :placeholder="field.placeholder"
          disabled
        >
      </div>
    </div>

    <!-- 未知 kind：预检已 fail closed，这里只保证不渲染半成品。 -->
    <p v-else class="text-sm text-muted" data-schema-unsupported>
      {{ description }}
    </p>
  </section>
</template>
