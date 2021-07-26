/******/ (() => { // webpackBootstrap
var __webpack_exports__ = {};
/*!*******************************************!*\
  !*** ./resources/js/plugins/Core/user.js ***!
  \*******************************************/
if (document.getElementById("vue-user-my-setting")) {
  var vums = {
    data: function data() {
      return {
        username: "",
        email: "",
        old_pwd: "",
        new_pwd: "",
        avatar: ""
      };
    },
    methods: {
      submit: function submit() {
        console.log(this.avatar);
      }
    },
    mounted: function mounted() {
      this.username = "aaa";
    }
  };
  Vue.createApp(vums).mount("#vue-user-my-setting");
}
/******/ })()
;