/******/ (() => { // webpackBootstrap
var __webpack_exports__ = {};
/*!*********************************************!*\
  !*** ./resources/js/plugins/Topic/topic.js ***!
  \*********************************************/
if (document.getElementById('topic-create')) {
  document.addEventListener("DOMContentLoaded", function () {
    var el;
    window.TomSelect && new TomSelect(el = document.getElementById('select-topic-tags'), {
      copyClassesToDropdown: false,
      dropdownClass: 'dropdown-menu ts-dropdown',
      optionClass: 'dropdown-item',
      controlInput: '<input>',
      render: {
        item: function item(data, escape) {
          if (data.customProperties) {
            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
          }

          return '<div>' + escape(data.text) + '</div>';
        },
        option: function option(data, escape) {
          if (data.customProperties) {
            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
          }

          return '<div>' + escape(data.text) + '</div>';
        }
      }
    });
  });
}
/******/ })()
;