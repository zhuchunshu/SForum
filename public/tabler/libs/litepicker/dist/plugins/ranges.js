/*!
 * 
 * plugins/ranges.js
 * Litepicker v2.0.12 (https://github.com/wakirin/Litepicker)
 * Package: litepicker (https://www.npmjs.com/package/litepicker)
 * License: MIT (https://github.com/wakirin/Litepicker/blob/master/LICENCE.md)
 * Copyright 2019-2021 Rinat G.
 *     
 * Hash: b9a648207aabe31b2912
 * 
 */!function(e){var n={};function t(s){if(n[s])return n[s].exports;var o=n[s]={i:s,l:!1,exports:{}};return e[s].call(o.exports,o,o.exports,t),o.l=!0,o.exports}t.m=e,t.c=n,t.d=function(e,n,s){t.o(e,n)||Object.defineProperty(e,n,{enumerable:!0,get:s})},t.r=function(e){"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},t.t=function(e,n){if(1&n&&(e=t(e)),8&n)return e;if(4&n&&"object"==typeof e&&e&&e.__esModule)return e;var o,s=Object.create(null);if(t.r(s),Object.defineProperty(s,"default",{enumerable:!0,value:e}),2&n&&"string"!=typeof e)for(o in e)t.d(s,o,function(t){return e[t]}.bind(null,o));return s},t.n=function(e){var n=e&&e.__esModule?function(){return e.default}:function(){return e};return t.d(n,"a",n),n},t.o=function(e,t){return Object.prototype.hasOwnProperty.call(e,t)},t.p="",t(t.s=8)}([function(e){"use strict";e.exports=function(e){var t=[];return t.toString=function(){return this.map(function(t){var n=function(e,t){var o,i,a,r,c,s=e[1]||"",n=e[3];return n?t&&"function"==typeof btoa?(o=(a=n,r=btoa(unescape(encodeURIComponent(JSON.stringify(a)))),c="sourceMappingURL=data:application/json;charset=utf-8;base64,".concat(r),"/*# ".concat(c," */")),i=n.sources.map(function(e){return"/*# sourceURL=".concat(n.sourceRoot||"").concat(e," */")}),[s].concat(i).concat([o]).join(`
`)):[s].join(`
`):s}(t,e);return t[2]?"@media ".concat(t[2]," {").concat(n,"}"):n}).join("")},t.i=function(e,n,s){"string"==typeof e&&(e=[[null,e,""]]);var o,i,a,r,c={};if(s)for(i=0;i<this.length;i++)r=this[i][0],r!=null&&(c[r]=!0);for(a=0;a<e.length;a++)o=[].concat(e[a]),s&&c[o[0]]||(n&&(o[2]?o[2]="".concat(n," and ").concat(o[2]):o[2]=n),t.push(o))},t}},function(e,t,n){"use strict";var o,i,a,u,h,s={},g=function(){return void 0===a&&(a=Boolean(window&&document&&document.all&&!window.atob)),a},f=function(){var e={};return function(t){if(void 0===e[t]){var n=document.querySelector(t);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch{n=null}e[t]=n}return e[t]}}();function d(e,t){for(var a=[],o={},i=0;i<e.length;i++){var n=e[i],s=t.base?n[0]+t.base:n[0],r={css:n[1],media:n[2],sourceMap:n[3]};o[s]?o[s].parts.push(r):a.push(o[s]={id:s,parts:[r]})}return a}function c(e,t){for(a=0;a<e.length;a++){var a,r,o=e[a],i=s[o.id],n=0;if(i){for(i.refs++;n<i.parts.length;n++)i.parts[n](o.parts[n]);for(;n<o.parts.length;n++)i.parts.push(m(o.parts[n],t))}else{for(r=[];n<o.parts.length;n++)r.push(m(o.parts[n],t));s[o.id]={id:o.id,refs:1,parts:r}}}}function l(e){var s,o,t=document.createElement("style");if(void 0===e.attributes.nonce&&(s=n.nc,s&&(e.attributes.nonce=s)),Object.keys(e.attributes).forEach(function(n){t.setAttribute(n,e.attributes[n])}),"function"==typeof e.insert)e.insert(t);else{if(o=f(e.insert||"head"),!o)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");o.appendChild(t)}return t}u=(o=[],function(e,t){return o[e]=t,o.filter(Boolean).join(`
`)});function r(e,t,n,s){if(i=n?"":s.css,e.styleSheet)e.styleSheet.cssText=u(t,i);else{var i,a=document.createTextNode(i),o=e.childNodes;o[t]&&e.removeChild(o[t]),o.length?e.insertBefore(a,o[t]):e.appendChild(a)}}function p(e,t,n){var s=n.css,o=n.media,i=n.sourceMap;if(o&&e.setAttribute("media",o),i&&btoa&&(s+=`
/*# sourceMappingURL=data:application/json;base64,`.concat(btoa(unescape(encodeURIComponent(JSON.stringify(i))))," */")),e.styleSheet)e.styleSheet.cssText=s;else{for(;e.firstChild;)e.removeChild(e.firstChild);e.appendChild(document.createTextNode(s))}}i=null,h=0;function m(e,t){if(t.singleton){var n,s,o,a=h++;n=i||(i=l(t)),s=r.bind(null,n,a,!1),o=r.bind(null,n,a,!0)}else n=l(t),s=p.bind(null,n,t),o=function(){!function(e){if(null===e.parentNode)return!1;e.parentNode.removeChild(e)}(n)};return s(e),function(t){if(t){if(t.css===e.css&&t.media===e.media&&t.sourceMap===e.sourceMap)return;s(e=t)}else o()}}e.exports=function(e,t){(t=t||{}).attributes="object"==typeof t.attributes?t.attributes:{},t.singleton||"boolean"==typeof t.singleton||(t.singleton=g());var n=d(e,t);return c(n,t),function(e){for(var o,i,a,r,h,l=[],u=0;u<n.length;u++)h=n[u],i=s[h.id],i&&(i.refs--,l.push(i));e&&c(d(e,t),t);for(a=0;a<l.length;a++)if(o=l[a],0===o.refs){for(r=0;r<o.parts.length;r++)o.parts[r]();delete s[o.id]}}}},,,,,,,function(e,t,n){"use strict";n.r(t),n(9);function o(e,t){var n,s=Object.keys(e);return Object.getOwnPropertySymbols&&(n=Object.getOwnPropertySymbols(e),t&&(n=n.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),s.push.apply(s,n)),s}function i(e){for(var t,n=1;n<arguments.length;n++)t=null!=arguments[n]?arguments[n]:{},n%2?o(Object(t),!0).forEach(function(n){s(e,n,t[n])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):o(Object(t)).forEach(function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))});return e}function s(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}Litepicker.add("ranges",{init:function(e){var t,n,o,a={position:"left",customRanges:{},customRangesLabels:["Today","Yesterday","Last 7 Days","Last 30 Days","This Month","Last Month"],force:!1,autoApply:e.options.autoApply};e.options.ranges=i(i({},a),e.options.ranges),e.options.singleMode=!1,t=e.options.ranges,Object.keys(t.customRanges).length||(n=e.DateTime(),t.customRanges=(s(o={},t.customRangesLabels[0],[n.clone(),n.clone()]),s(o,t.customRangesLabels[1],[n.clone().subtract(1,"day"),n.clone().subtract(1,"day")]),s(o,t.customRangesLabels[2],[n.clone().subtract(6,"day"),n]),s(o,t.customRangesLabels[3],[n.clone().subtract(29,"day"),n]),s(o,t.customRangesLabels[4],function(e){var t=e.clone();return t.setDate(1),[t,new Date(e.getFullYear(),e.getMonth()+1,0)]}(n)),s(o,t.customRangesLabels[5],function(e){var t=e.clone();return t.setDate(1),t.setMonth(e.getMonth()-1),[t,new Date(e.getFullYear(),e.getMonth(),0)]}(n)),o)),e.on("render",function(n){var s=document.createElement("div");s.className="container__predefined-ranges",e.ui.dataset.rangesPosition=t.position,Object.keys(t.customRanges).forEach(function(o){var a=t.customRanges[o],i=document.createElement("button");i.innerText=o,i.tabIndex=n.dataset.plugins.indexOf("keyboardnav")>=0?1:-1,i.dataset.start=a[0].getTime(),i.dataset.end=a[1].getTime(),i.addEventListener("click",function(n){if(o=n.target,o){var o,s=e.DateTime(Number(o.dataset.start)),i=e.DateTime(Number(o.dataset.end));t.autoApply?(e.setDateRange(s,i,t.force),e.emit("ranges.selected",s,i),e.hide()):(e.datePicked=[s,i],e.emit("ranges.preselect",s,i)),!e.options.inlineMode&&t.autoApply||e.gotoDate(s)}}),s.appendChild(i)}),n.querySelector(".container__main").prepend(s)})}})},function(e,t,n){var o,s=n(10);"string"==typeof s&&(s=[[e.i,s,""]]),o={insert:function(e){var t=document.querySelector("head"),n=window._lastElementInsertedByStyleLoader;window.disableLitepickerStyles||(n?n.nextSibling?t.insertBefore(e,n.nextSibling):t.appendChild(e):t.insertBefore(e,t.firstChild),window._lastElementInsertedByStyleLoader=e)},singleton:!1},n(1)(s,o),s.locals&&(e.exports=s.locals)},function(e,t,n){(t=n(0)(!1)).push([e.i,`.litepicker[data-plugins*="ranges"] > .container__main > .container__predefined-ranges {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  background: var(--litepicker-container-months-color-bg);
  box-shadow: -2px 0px 5px var(--litepicker-footer-box-shadow-color);
  border-radius: 3px;
}
.litepicker[data-plugins*="ranges"][data-ranges-position="left"] > .container__main {
  /* */
}
.litepicker[data-plugins*="ranges"][data-ranges-position="right"] > .container__main{
  flex-direction: row-reverse;
}
.litepicker[data-plugins*="ranges"][data-ranges-position="right"] > .container__main > .container__predefined-ranges {
  box-shadow: 2px 0px 2px var(--litepicker-footer-box-shadow-color);
}
.litepicker[data-plugins*="ranges"][data-ranges-position="top"] > .container__main {
  flex-direction: column;
}
.litepicker[data-plugins*="ranges"][data-ranges-position="top"] > .container__main > .container__predefined-ranges {
  flex-direction: row;
  box-shadow: 2px 0px 2px var(--litepicker-footer-box-shadow-color);
}
.litepicker[data-plugins*="ranges"][data-ranges-position="bottom"] > .container__main {
  flex-direction: column-reverse;
}
.litepicker[data-plugins*="ranges"][data-ranges-position="bottom"] > .container__main > .container__predefined-ranges {
  flex-direction: row;
  box-shadow: 2px 0px 2px var(--litepicker-footer-box-shadow-color);
}
.litepicker[data-plugins*="ranges"] > .container__main > .container__predefined-ranges button {
  padding: 5px;
  margin: 2px 0;
}
.litepicker[data-plugins*="ranges"][data-ranges-position="left"] > .container__main > .container__predefined-ranges button,
.litepicker[data-plugins*="ranges"][data-ranges-position="right"] > .container__main > .container__predefined-ranges button{
  width: 100%;
  text-align: left;
}
.litepicker[data-plugins*="ranges"] > .container__main > .container__predefined-ranges button:hover {
  cursor: default;
  opacity: .6;
}`,""]),e.exports=t}])