/*!
* Tabler v1.0.0-beta19 (https://tabler.io)
* @version 1.0.0-beta19
* @link https://tabler.io
* Copyright 2018-2023 The Tabler Authors
* Copyright 2018-2023 codecalm.net Paweł Kuna
* Licensed under MIT (https://github.com/tabler/tabler/blob/master/LICENSE)
*/(function(e){typeof define=="function"&&define.amd?define(e):e()})(function(){"use strict";var e,n,s="tablerTheme",o="light",t=new Proxy(new URLSearchParams(window.location.search),{get:function(t,n){return t.get(n)}});t.theme?(localStorage.setItem(s,t.theme),e=t.theme):(n=localStorage.getItem(s),e=n||o),e==="dark"?document.body.setAttribute("data-bs-theme",e):document.body.removeAttribute("data-bs-theme")})