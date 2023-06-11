/*!
 * 
 * plugins/mobilefriendly.js
 * Litepicker v2.0.12 (https://github.com/wakirin/Litepicker)
 * Package: litepicker (https://www.npmjs.com/package/litepicker)
 * License: MIT (https://github.com/wakirin/Litepicker/blob/master/LICENCE.md)
 * Copyright 2019-2021 Rinat G.
 *     
 * Hash: b9a648207aabe31b2912
 * 
 */!function(e){var n={};function t(s){if(n[s])return n[s].exports;var o=n[s]={i:s,l:!1,exports:{}};return e[s].call(o.exports,o,o.exports,t),o.l=!0,o.exports}t.m=e,t.c=n,t.d=function(e,n,s){t.o(e,n)||Object.defineProperty(e,n,{enumerable:!0,get:s})},t.r=function(e){"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},t.t=function(e,n){if(1&n&&(e=t(e)),8&n)return e;if(4&n&&"object"==typeof e&&e&&e.__esModule)return e;var o,s=Object.create(null);if(t.r(s),Object.defineProperty(s,"default",{enumerable:!0,value:e}),2&n&&"string"!=typeof e)for(o in e)t.d(s,o,function(t){return e[t]}.bind(null,o));return s},t.n=function(e){var n=e&&e.__esModule?function(){return e.default}:function(){return e};return t.d(n,"a",n),n},t.o=function(e,t){return Object.prototype.hasOwnProperty.call(e,t)},t.p="",t(t.s=5)}([function(e){"use strict";e.exports=function(e){var t=[];return t.toString=function(){return this.map(function(t){var n=function(e,t){var o,i,a,r,c,s=e[1]||"",n=e[3];return n?t&&"function"==typeof btoa?(o=(a=n,r=btoa(unescape(encodeURIComponent(JSON.stringify(a)))),c="sourceMappingURL=data:application/json;charset=utf-8;base64,".concat(r),"/*# ".concat(c," */")),i=n.sources.map(function(e){return"/*# sourceURL=".concat(n.sourceRoot||"").concat(e," */")}),[s].concat(i).concat([o]).join(`
`)):[s].join(`
`):s}(t,e);return t[2]?"@media ".concat(t[2]," {").concat(n,"}"):n}).join("")},t.i=function(e,n,s){"string"==typeof e&&(e=[[null,e,""]]);var o,i,a,r,c={};if(s)for(i=0;i<this.length;i++)r=this[i][0],r!=null&&(c[r]=!0);for(a=0;a<e.length;a++)o=[].concat(e[a]),s&&c[o[0]]||(n&&(o[2]?o[2]="".concat(n," and ").concat(o[2]):o[2]=n),t.push(o))},t}},function(e,t,n){"use strict";var o,i,a,u,h,s={},g=function(){return void 0===a&&(a=Boolean(window&&document&&document.all&&!window.atob)),a},f=function(){var e={};return function(t){if(void 0===e[t]){var n=document.querySelector(t);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch{n=null}e[t]=n}return e[t]}}();function d(e,t){for(var a=[],o={},i=0;i<e.length;i++){var n=e[i],s=t.base?n[0]+t.base:n[0],r={css:n[1],media:n[2],sourceMap:n[3]};o[s]?o[s].parts.push(r):a.push(o[s]={id:s,parts:[r]})}return a}function c(e,t){for(a=0;a<e.length;a++){var a,r,o=e[a],i=s[o.id],n=0;if(i){for(i.refs++;n<i.parts.length;n++)i.parts[n](o.parts[n]);for(;n<o.parts.length;n++)i.parts.push(m(o.parts[n],t))}else{for(r=[];n<o.parts.length;n++)r.push(m(o.parts[n],t));s[o.id]={id:o.id,refs:1,parts:r}}}}function l(e){var s,o,t=document.createElement("style");if(void 0===e.attributes.nonce&&(s=n.nc,s&&(e.attributes.nonce=s)),Object.keys(e.attributes).forEach(function(n){t.setAttribute(n,e.attributes[n])}),"function"==typeof e.insert)e.insert(t);else{if(o=f(e.insert||"head"),!o)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");o.appendChild(t)}return t}u=(o=[],function(e,t){return o[e]=t,o.filter(Boolean).join(`
`)});function r(e,t,n,s){if(i=n?"":s.css,e.styleSheet)e.styleSheet.cssText=u(t,i);else{var i,a=document.createTextNode(i),o=e.childNodes;o[t]&&e.removeChild(o[t]),o.length?e.insertBefore(a,o[t]):e.appendChild(a)}}function p(e,t,n){var s=n.css,o=n.media,i=n.sourceMap;if(o&&e.setAttribute("media",o),i&&btoa&&(s+=`
/*# sourceMappingURL=data:application/json;base64,`.concat(btoa(unescape(encodeURIComponent(JSON.stringify(i))))," */")),e.styleSheet)e.styleSheet.cssText=s;else{for(;e.firstChild;)e.removeChild(e.firstChild);e.appendChild(document.createTextNode(s))}}i=null,h=0;function m(e,t){if(t.singleton){var n,s,o,a=h++;n=i||(i=l(t)),s=r.bind(null,n,a,!1),o=r.bind(null,n,a,!0)}else n=l(t),s=p.bind(null,n,t),o=function(){!function(e){if(null===e.parentNode)return!1;e.parentNode.removeChild(e)}(n)};return s(e),function(t){if(t){if(t.css===e.css&&t.media===e.media&&t.sourceMap===e.sourceMap)return;s(e=t)}else o()}}e.exports=function(e,t){(t=t||{}).attributes="object"==typeof t.attributes?t.attributes:{},t.singleton||"boolean"==typeof t.singleton||(t.singleton=g());var n=d(e,t);return c(n,t),function(e){for(var o,i,a,r,h,l=[],u=0;u<n.length;u++)h=n[u],i=s[h.id],i&&(i.refs--,l.push(i));e&&c(d(e,t),t);for(a=0;a<l.length;a++)if(o=l[a],0===o.refs){for(r=0;r<o.parts.length;r++)o.parts[r]();delete s[o.id]}}}},,,,function(e,t,n){"use strict";n.r(t),n(6);function s(e,t){var n,s=Object.keys(e);return Object.getOwnPropertySymbols&&(n=Object.getOwnPropertySymbols(e),t&&(n=n.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),s.push.apply(s,n)),s}function o(e){for(var t,n=1;n<arguments.length;n++)t=null!=arguments[n]?arguments[n]:{},n%2?s(Object(t),!0).forEach(function(n){i(e,n,t[n])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):s(Object(t)).forEach(function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))});return e}function i(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}Litepicker.add("mobilefriendly",{init:function(e){t=e.options,e.options.mobilefriendly=o(o({},{breakpoint:480}),t.mobilefriendly),Object.defineProperties(e,{xTouchDown:{value:null,writable:!0},yTouchDown:{value:null,writable:!0},touchTargetMonth:{value:null,writable:!0}}),n=!1;try{a=Object.defineProperty({},"passive",{get:function(){n=!0}}),window.addEventListener("testPassive",null,a),window.removeEventListener("testPassive",null,a)}catch{}function s(){var t="portrait"===i();return window.matchMedia("(max-device-".concat(t?"width":"height",": ").concat(e.options.mobilefriendly.breakpoint,"px)")).matches}function i(){return"orientation"in window.screen&&"type"in window.screen.orientation?window.screen.orientation.type.replace(/-\w+$/,""):window.matchMedia("(orientation: portrait)").matches?"portrait":"landscape"}function r(){"portrait"===i()?(e.options.numberOfMonths=1,e.options.numberOfColumns=1):(e.options.numberOfMonths=2,e.options.numberOfColumns=2)}var t,n,a,c=function(t){var n=t.touches[0];e.xTouchDown=n.clientX,e.yTouchDown=n.clientY},l=function(t){if(e.xTouchDown&&e.yTouchDown){var i,a,p=t.touches[0].clientX,g=t.touches[0].clientY,n=e.xTouchDown-p,l=e.yTouchDown-g,h=Math.abs(n)>Math.abs(l),d=e.options.numberOfMonths,s=null,o=!1,c="",m=Array.from(e.ui.querySelectorAll(".month-item"));if(h){var u=e.DateTime(e.ui.querySelector(".day-item").dataset.time),f=Number("".concat(1-Math.abs(n)/100)),r=0;n>0?(r=-Math.abs(n),s=u.clone().add(d,"month"),a=e.options.maxDate,o=!a||s.isSameOrBefore(e.DateTime(a),"month"),c="next"):(r=Math.abs(n),s=u.clone().subtract(d,"month"),i=e.options.minDate,o=!i||s.isSameOrAfter(e.DateTime(i),"month"),c="prev"),o&&m.map(function(e){e.style.opacity=f,e.style.transform="translateX(".concat(r,"px)")})}Math.abs(n)+Math.abs(l)>100&&h&&s&&o&&(e.touchTargetMonth=c,e.gotoDate(s))}},d=function(){e.touchTargetMonth||Array.from(e.ui.querySelectorAll(".month-item")).map(function(e){e.style.transform="translateX(0px)",e.style.opacity=1}),e.xTouchDown=null,e.yTouchDown=null};e.backdrop=document.createElement("div"),e.backdrop.className="litepicker-backdrop",e.backdrop.addEventListener("click",e.hide()),t.element&&t.element.parentNode&&t.element.parentNode.appendChild(e.backdrop),window.addEventListener("orientationchange",function(){window.addEventListener("resize",function n(){if(s()&&e.isShowning()){var o=i();switch(o){case"landscape":t.numberOfMonths=2,t.numberOfColumns=2;break;default:t.numberOfMonths=1,t.numberOfColumns=1}e.ui.classList.toggle("mobilefriendly-portrait","portrait"===o),e.ui.classList.toggle("mobilefriendly-landscape","landscape"===o),e.render()}window.removeEventListener("resize",n)})}),t.inlineMode&&s()&&(window.dispatchEvent(new Event("orientationchange")),window.dispatchEvent(new Event("resize"))),e.on("before:show",function(t){if(e.triggerElement=t,!e.options.inlineMode&&s()){e.emit("mobilefriendly.before:show",t),e.ui.style.position="fixed",e.ui.style.display="block",r(),e.scrollToDate(t),e.render();var n=i();e.ui.classList.add("mobilefriendly"),e.ui.classList.toggle("mobilefriendly-portrait","portrait"===n),e.ui.classList.toggle("mobilefriendly-landscape","landscape"===n),e.ui.style.top="50%",e.ui.style.left="50%",e.ui.style.right=null,e.ui.style.bottom=null,e.ui.style.zIndex=e.options.zIndex,e.backdrop.style.display="block",e.backdrop.style.zIndex=e.options.zIndex-1,document.body.classList.add("litepicker-open"),(t||e.options.element).blur(),e.emit("mobilefriendly.show",t)}else s()&&(r(),e.render())}),e.on("render",function(){e.touchTargetMonth&&Array.from(e.ui.querySelectorAll(".month-item")).map(function(t){return t.classList.add("touch-target-".concat(e.touchTargetMonth))}),e.touchTargetMonth=null}),e.on("hide",function(){document.body.classList.remove("litepicker-open"),e.backdrop.style.display="none",e.ui.classList.remove("mobilefriendly","mobilefriendly-portrait","mobilefriendly-landscape")}),e.on("destroy",function(){e.backdrop&&e.backdrop.parentNode&&e.backdrop.parentNode.removeChild(e.backdrop)}),e.ui.addEventListener("touchstart",c,!!n&&{passive:!0}),e.ui.addEventListener("touchmove",l,!!n&&{passive:!0}),e.ui.addEventListener("touchend",d,!!n&&{passive:!0})}})},function(e,t,n){var o,s=n(7);"string"==typeof s&&(s=[[e.i,s,""]]),o={insert:function(e){var t=document.querySelector("head"),n=window._lastElementInsertedByStyleLoader;window.disableLitepickerStyles||(n?n.nextSibling?t.insertBefore(e,n.nextSibling):t.appendChild(e):t.insertBefore(e,t.firstChild),window._lastElementInsertedByStyleLoader=e)},singleton:!1},n(1)(s,o),s.locals&&(e.exports=s.locals)},function(e,t,n){(t=n(0)(!1)).push([e.i,`:root {
  --litepicker-mobilefriendly-backdrop-color-bg: #000;
}

.litepicker-backdrop {
  display: none;
  background-color: var(--litepicker-mobilefriendly-backdrop-color-bg);
  opacity: 0.3;
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
}

.litepicker-open {
  overflow: hidden;
}

.litepicker.mobilefriendly[data-plugins*="mobilefriendly"] {
  transform: translate(-50%, -50%);
  font-size: 1.1rem;
  --litepicker-container-months-box-shadow-color: #616161;
}
.litepicker.mobilefriendly-portrait {
  --litepicker-day-width: 13.5vw;
  --litepicker-month-width: calc(var(--litepicker-day-width) * 7);
}
.litepicker.mobilefriendly-landscape {
  --litepicker-day-width: 5.5vw;
  --litepicker-month-width: calc(var(--litepicker-day-width) * 7);
}

.litepicker[data-plugins*="mobilefriendly"] .container__months {
  overflow: hidden;
}

.litepicker.mobilefriendly[data-plugins*="mobilefriendly"] .container__months .month-item-header {
  height: var(--litepicker-day-width);
}

.litepicker.mobilefriendly[data-plugins*="mobilefriendly"] .container__days > div {
  height: var(--litepicker-day-width);
  display: flex;
  align-items: center;
  justify-content: center;
}


.litepicker[data-plugins*="mobilefriendly"] .container__months .month-item {
  transform-origin: center;
}

.litepicker[data-plugins*="mobilefriendly"] .container__months .month-item.touch-target-next {
  animation-name: lp-bounce-target-next;
  animation-duration: .5s;
  animation-timing-function: ease;
}

.litepicker[data-plugins*="mobilefriendly"] .container__months .month-item.touch-target-prev {
  animation-name: lp-bounce-target-prev;
  animation-duration: .5s;
  animation-timing-function: ease;
}

@keyframes lp-bounce-target-next {
  from {
    transform: translateX(100px) scale(0.5);
  }
  to {
    transform: translateX(0px) scale(1);
  }
}

@keyframes lp-bounce-target-prev {
  from {
    transform: translateX(-100px) scale(0.5);
  }
  to {
    transform: translateX(0px) scale(1);
  }
}`,""]),e.exports=t}])