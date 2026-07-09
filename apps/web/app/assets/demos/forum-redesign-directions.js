const directionTabs = [...document.querySelectorAll("[data-direction-tab]")]
const directions = [...document.querySelectorAll("[data-direction]")]

function setDirection(id) {
  directionTabs.forEach((tab) => tab.setAttribute("aria-pressed", String(tab.dataset.directionTab === id)))
  directions.forEach((direction) => direction.classList.toggle("is-active", direction.dataset.direction === id))
}

directionTabs.forEach((tab) => {
  tab.addEventListener("click", () => setDirection(tab.dataset.directionTab))
})

document.querySelectorAll(".direction").forEach((direction) => {
  const screenTabs = [...direction.querySelectorAll("[data-screen-tab]")]
  const screens = [...direction.querySelectorAll("[data-screen]")]
  screenTabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const screenName = tab.dataset.screenTab
      screenTabs.forEach((item) => item.setAttribute("aria-pressed", String(item === tab)))
      screens.forEach((screen) => screen.classList.toggle("is-active", screen.dataset.screen === screenName))
    })
  })
})
