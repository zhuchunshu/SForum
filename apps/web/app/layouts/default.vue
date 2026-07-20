<template>
  <!--
    公开 chrome 由主题 L1（sf-navbar/sf-footer）或 SFPageOutlet fail-closed 宿主壳拥有。
    Nuxt default layout 仅透传，避免与主题双层导航。
  -->
  <slot />
</template>
