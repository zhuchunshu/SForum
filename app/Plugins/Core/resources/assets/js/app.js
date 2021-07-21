/******/ (() => { // webpackBootstrap
var __webpack_exports__ = {};
/*!******************************************!*\
  !*** ./resources/js/plugins/Core/app.js ***!
  \******************************************/
var header = {
  data: function data() {
    return {
      center: true,
      flex: true,
      hidden: true,
      block: false
    };
  },
  methods: {
    menu: function menu() {
      if (this.hidden === true) {
        this.hidden = false;
        this.block = true;
        document.getElementById("app-mobile-menu").setAttribute("class", "items-center md:flex block");
      } else {
        this.hidden = true;
        this.block = false;
        document.getElementById("app-mobile-menu").setAttribute("class", "items-center md:flex hidden");
      }
    }
  }
};
Vue.createApp(header).mount("#app-header");
/******/ })()
;