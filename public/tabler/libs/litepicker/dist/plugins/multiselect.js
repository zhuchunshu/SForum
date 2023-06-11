/*!
 * 
 * plugins/multiselect.js
 * Litepicker v2.0.12 (https://github.com/wakirin/Litepicker)
 * Package: litepicker (https://www.npmjs.com/package/litepicker)
 * License: MIT (https://github.com/wakirin/Litepicker/blob/master/LICENCE.md)
 * Copyright 2019-2021 Rinat G.
 *     
 * Hash: b9a648207aabe31b2912
 * 
 */!function(e){var n={};function t(s){if(n[s])return n[s].exports;var o=n[s]={i:s,l:!1,exports:{}};return e[s].call(o.exports,o,o.exports,t),o.l=!0,o.exports}t.m=e,t.c=n,t.d=function(e,n,s){t.o(e,n)||Object.defineProperty(e,n,{enumerable:!0,get:s})},t.r=function(e){"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},t.t=function(e,n){if(1&n&&(e=t(e)),8&n)return e;if(4&n&&"object"==typeof e&&e&&e.__esModule)return e;var o,s=Object.create(null);if(t.r(s),Object.defineProperty(s,"default",{enumerable:!0,value:e}),2&n&&"string"!=typeof e)for(o in e)t.d(s,o,function(t){return e[t]}.bind(null,o));return s},t.n=function(e){var n=e&&e.__esModule?function(){return e.default}:function(){return e};return t.d(n,"a",n),n},t.o=function(e,t){return Object.prototype.hasOwnProperty.call(e,t)},t.p="",t(t.s=11)}([function(e){"use strict";e.exports=function(e){var t=[];return t.toString=function(){return this.map(function(t){var n=function(e,t){var o,i,a,r,c,s=e[1]||"",n=e[3];return n?t&&"function"==typeof btoa?(o=(a=n,r=btoa(unescape(encodeURIComponent(JSON.stringify(a)))),c="sourceMappingURL=data:application/json;charset=utf-8;base64,".concat(r),"/*# ".concat(c," */")),i=n.sources.map(function(e){return"/*# sourceURL=".concat(n.sourceRoot||"").concat(e," */")}),[s].concat(i).concat([o]).join(`
`)):[s].join(`
`):s}(t,e);return t[2]?"@media ".concat(t[2]," {").concat(n,"}"):n}).join("")},t.i=function(e,n,s){"string"==typeof e&&(e=[[null,e,""]]);var o,i,a,r,c={};if(s)for(i=0;i<this.length;i++)r=this[i][0],r!=null&&(c[r]=!0);for(a=0;a<e.length;a++)o=[].concat(e[a]),s&&c[o[0]]||(n&&(o[2]?o[2]="".concat(n," and ").concat(o[2]):o[2]=n),t.push(o))},t}},function(e,t,n){"use strict";var o,i,a,u,h,s={},g=function(){return void 0===a&&(a=Boolean(window&&document&&document.all&&!window.atob)),a},f=function(){var e={};return function(t){if(void 0===e[t]){var n=document.querySelector(t);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch{n=null}e[t]=n}return e[t]}}();function d(e,t){for(var a=[],o={},i=0;i<e.length;i++){var n=e[i],s=t.base?n[0]+t.base:n[0],r={css:n[1],media:n[2],sourceMap:n[3]};o[s]?o[s].parts.push(r):a.push(o[s]={id:s,parts:[r]})}return a}function c(e,t){for(a=0;a<e.length;a++){var a,r,o=e[a],i=s[o.id],n=0;if(i){for(i.refs++;n<i.parts.length;n++)i.parts[n](o.parts[n]);for(;n<o.parts.length;n++)i.parts.push(m(o.parts[n],t))}else{for(r=[];n<o.parts.length;n++)r.push(m(o.parts[n],t));s[o.id]={id:o.id,refs:1,parts:r}}}}function l(e){var s,o,t=document.createElement("style");if(void 0===e.attributes.nonce&&(s=n.nc,s&&(e.attributes.nonce=s)),Object.keys(e.attributes).forEach(function(n){t.setAttribute(n,e.attributes[n])}),"function"==typeof e.insert)e.insert(t);else{if(o=f(e.insert||"head"),!o)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");o.appendChild(t)}return t}u=(o=[],function(e,t){return o[e]=t,o.filter(Boolean).join(`
`)});function r(e,t,n,s){if(i=n?"":s.css,e.styleSheet)e.styleSheet.cssText=u(t,i);else{var i,a=document.createTextNode(i),o=e.childNodes;o[t]&&e.removeChild(o[t]),o.length?e.insertBefore(a,o[t]):e.appendChild(a)}}function p(e,t,n){var s=n.css,o=n.media,i=n.sourceMap;if(o&&e.setAttribute("media",o),i&&btoa&&(s+=`
/*# sourceMappingURL=data:application/json;base64,`.concat(btoa(unescape(encodeURIComponent(JSON.stringify(i))))," */")),e.styleSheet)e.styleSheet.cssText=s;else{for(;e.firstChild;)e.removeChild(e.firstChild);e.appendChild(document.createTextNode(s))}}i=null,h=0;function m(e,t){if(t.singleton){var n,s,o,a=h++;n=i||(i=l(t)),s=r.bind(null,n,a,!1),o=r.bind(null,n,a,!0)}else n=l(t),s=p.bind(null,n,t),o=function(){!function(e){if(null===e.parentNode)return!1;e.parentNode.removeChild(e)}(n)};return s(e),function(t){if(t){if(t.css===e.css&&t.media===e.media&&t.sourceMap===e.sourceMap)return;s(e=t)}else o()}}e.exports=function(e,t){(t=t||{}).attributes="object"==typeof t.attributes?t.attributes:{},t.singleton||"boolean"==typeof t.singleton||(t.singleton=g());var n=d(e,t);return c(n,t),function(e){for(var o,i,a,r,h,l=[],u=0;u<n.length;u++)h=n[u],i=s[h.id],i&&(i.refs--,l.push(i));e&&c(d(e,t),t);for(a=0;a<l.length;a++)if(o=l[a],0===o.refs){for(r=0;r<o.parts.length;r++)o.parts[r]();delete s[o.id]}}}},,,,,,,,,,function(e,t,n){"use strict";n.r(t),n(12);function o(e){return function(e){if(Array.isArray(e))return s(e)}(e)||function(e){if("undefined"!=typeof Symbol&&Symbol.iterator in Object(e))return Array.from(e)}(e)||function(e,t){if(!e)return;if("string"==typeof e)return s(e,t);var n=Object.prototype.toString.call(e).slice(8,-1);if("Object"===n&&e.constructor&&(n=e.constructor.name),"Map"===n||"Set"===n)return Array.from(e);if("Arguments"===n||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n))return s(e,t)}(e)||function(){throw new TypeError(`Invalid attempt to spread non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)}()}function s(e,t){(t==null||t>e.length)&&(t=e.length);for(var n=0,s=new Array(t);n<t;n++)s[n]=e[n];return s}function i(e,t){var n,s=Object.keys(e);return Object.getOwnPropertySymbols&&(n=Object.getOwnPropertySymbols(e),t&&(n=n.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),s.push.apply(s,n)),s}function a(e){for(var t,n=1;n<arguments.length;n++)t=null!=arguments[n]?arguments[n]:{},n%2?i(Object(t),!0).forEach(function(n){r(e,n,t[n])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach(function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))});return e}function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}Litepicker.add("multiselect",{init:function(e){Object.defineProperties(e,{multipleDates:{value:[],enumerable:!0,writable:!0},preMultipleDates:{value:[],writable:!0}}),e.options.multiselect=a(a({},{max:null}),e.options.multiselect),e.options.autoApply=e.options.inlineMode,e.options.showTooltip=!1;var t=function(){if(t=e.preMultipleDates.length,n=e.ui.querySelector(".preview-date-range"),n&&t>0){var t,n,s=e.pluralSelector(t),o=e.options.tooltipText[s]?e.options.tooltipText[s]:"[".concat(s,"]"),i="".concat(t," ").concat(o);n.innerText=i}};e.on("before:show",function(){e.preMultipleDates=o(e.multipleDates)}),e.on("show",function(){t()}),e.on("before:click",function(n){if(n.classList.contains("day-item")){if(e.preventClick=!0,n.classList.contains("is-locked"))return void n.blur();var s=Number(n.dataset.time);n.classList.contains("is-selected")?(e.preMultipleDates=e.preMultipleDates.filter(function(e){return e!==s}),e.emit("multiselect.deselect",e.DateTime(s))):(e.preMultipleDates[e.preMultipleDates.length]=s,e.emit("multiselect.select",e.DateTime(s))),e.options.autoApply&&e.emit("button:apply"),e.render(),t()}}),e.on("render:day",function(t){var s=e.preMultipleDates.filter(function(e){return e===Number(t.dataset.time)}).length,n=Number(e.options.multiselect.max);s?t.classList.add("is-selected"):n&&e.preMultipleDates.length>=n&&t.classList.add("is-locked")}),e.on("button:cancel",function(){e.preMultipleDates.length=0}),e.on("button:apply",function(){e.multipleDates=o(e.preMultipleDates).sort(function(e,t){return e-t})}),e.on("clear:selection",function(){e.clearMultipleDates(),e.render()}),e.clearMultipleDates=function(){e.preMultipleDates.length=0,e.multipleDates.length=0},e.getMultipleDates=function(){return e.multipleDates.map(function(t){return e.DateTime(t)})},e.multipleDatesToString=function(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:"YYYY-MM-DD",n=arguments.length>1&&void 0!==arguments[1]?arguments[1]:",";return e.multipleDates.map(function(n){return e.DateTime(n).format(t)}).join(n)}}})},function(e,t,n){var o,s=n(13);"string"==typeof s&&(s=[[e.i,s,""]]),o={insert:function(e){var t=document.querySelector("head"),n=window._lastElementInsertedByStyleLoader;window.disableLitepickerStyles||(n?n.nextSibling?t.insertBefore(e,n.nextSibling):t.appendChild(e):t.insertBefore(e,t.firstChild),window._lastElementInsertedByStyleLoader=e)},singleton:!1},n(1)(s,o),s.locals&&(e.exports=s.locals)},function(e,t,n){(t=n(0)(!1)).push([e.i,`:root {
  --litepicker-multiselect-is-selected-color-bg: #2196f3;
  --litepicker-multiselect-is-selected-color: #fff;
  --litepicker-multiselect-hover-color-bg: #2196f3;
  --litepicker-multiselect-hover-color: #fff;
}

.litepicker[data-plugins*="multiselect"] .container__days .day-item {
  position: relative;
  z-index: 1;
}

.litepicker[data-plugins*="multiselect"] .container__days .day-item:not(.is-locked):after {
  content: '';
  position: absolute;
  width: 27px;
  height: 27px;
  top: 50%;
  left: 50%;
  z-index: -1;
  border-radius: 50%;
  transform: translate(-50%, -50%);
}

.litepicker[data-plugins*="multiselect"] .container__days .day-item:not(.is-locked):hover {
  box-shadow: none;
  color: var(--litepicker-day-color);
  font-weight: bold;
}


.litepicker[data-plugins*="multiselect"] .container__days .day-item.is-selected,
.litepicker[data-plugins*="multiselect"] .container__days .day-item.is-selected:hover {
  color: var(--litepicker-multiselect-is-selected-color);
}

.litepicker[data-plugins*="multiselect"] .container__days .day-item.is-selected:after {
  color: var(--litepicker-multiselect-is-selected-color);
  background-color: var(--litepicker-multiselect-is-selected-color-bg);
}
`,""]),e.exports=t}])