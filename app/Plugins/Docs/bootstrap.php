<?php

// 后台菜单
menu()->add(701,[
    'url' => '/admin/docs',
    'name' => '文档设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-note" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="13" y1="20" x2="20" y2="13"></line>
   <path d="M13 20v-6a1 1 0 0 1 1 -1h6v-7a2 2 0 0 0 -2 -2h-12a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7"></path>
</svg>',
]);

// 用户组权限
Authority()->add("docs_create","创建文档");
Authority()->add("docs_edit","修改自己的文档");
Authority()->add("docs_delete","删除自己的文档");
Authority()->add("admin_docs_edit","修改所有文档");
Authority()->add("admin_docs_delete","删除所有文档");

// 首页菜单
Itf()->add("menu",701,[
    "name" => "本站文档",
    "url" => "/docs",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-note" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="13" y1="20" x2="20" y2="13"></line>
   <path d="M13 20v-6a1 1 0 0 1 1 -1h6v-7a2 2 0 0 0 -2 -2h-12a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7"></path>
</svg>',
]);