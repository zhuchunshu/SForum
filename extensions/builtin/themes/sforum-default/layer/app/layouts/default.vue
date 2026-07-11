<template>
  <div class="flex flex-col min-h-screen">
    <SFNavbar />
    <SFAnnouncementBanner />
    <div class="flex-1">
      <slot />
    </div>
    <SFFooter />
  </div>
</template>
