<script setup lang="ts">
type InputSize = 'sm' | 'md' | 'lg'
type InputType = 'email' | 'number' | 'password' | 'search' | 'text' | 'url'

const props = withDefaults(defineProps<{
  modelValue?: string | number
  label?: string
  hint?: string
  error?: string
  id?: string
  name?: string
  placeholder?: string
  type?: InputType
  size?: InputSize
  disabled?: boolean
  required?: boolean
}>(), {
  modelValue: '',
  label: undefined,
  hint: undefined,
  error: undefined,
  id: undefined,
  name: undefined,
  placeholder: undefined,
  type: 'text',
  size: 'md',
  disabled: false,
  required: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const generatedId = useId()
const inputId = computed(() => props.id || generatedId)
const message = computed(() => props.error || props.hint)
const inputClass = computed(() => [
  'sf-input',
  props.error ? 'sf-input--invalid' : ''
].filter(Boolean).join(' '))
const controlClass = computed(() => [
  'sf-input__control',
  `sf-input__control--${props.size}`
].join(' '))

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div :class="inputClass">
    <label v-if="label" class="sf-input__label" :for="inputId">
      {{ label }}
    </label>
    <input
      :id="inputId"
      :class="controlClass"
      :name="name"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :required="required"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="message ? `${inputId}-message` : undefined"
      @input="onInput"
    >
    <p v-if="message" :id="`${inputId}-message`" class="sf-input__message">
      {{ message }}
    </p>
  </div>
</template>
