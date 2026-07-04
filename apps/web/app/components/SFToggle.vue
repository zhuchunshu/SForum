<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue?: boolean
  label?: string
  description?: string
  disabled?: boolean
}>(), {
  modelValue: false,
  label: undefined,
  description: undefined,
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
  }
}
</script>

<template>
  <div class="sf-toggle">
    <button
      type="button"
      class="sf-toggle__button"
      role="switch"
      :aria-checked="modelValue ? 'true' : 'false'"
      :disabled="disabled"
      @click="toggle"
    />
    <div>
      <p v-if="label" class="sf-toggle__label">
        {{ label }}
      </p>
      <p v-if="description" class="sf-toggle__description">
        {{ description }}
      </p>
    </div>
  </div>
</template>
