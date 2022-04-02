<?php
// 仪表盘
menu()->add(0,[
    'url' => '/admin',
    'name' => '仪表盘',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-dashboard" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <circle cx="12" cy="13" r="2"></circle>
    <line x1="13.45" y1="11.55" x2="15.5" y2="9.5"></line>
    <path d="M6.4 20a9 9 0 1 1 11.2 0z"></path>
 </svg>',
]);
menu()->add(1,[
    'url' => '/admin/plugins',
    'name' => '组件',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-package" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
    <line x1="12" y1="12" x2="20" y2="7.5"></line>
    <line x1="12" y1="12" x2="12" y2="21"></line>
    <line x1="12" y1="12" x2="4" y2="7.5"></line>
    <line x1="16" y1="5.25" x2="8" y2="9.75"></line>
 </svg>',
]);
menu()->add(6,[
	'url' => '/admin/themes',
	'name' => '主题',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-template" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <rect x="4" y="4" width="16" height="4" rx="1"></rect>
   <rect x="4" y="12" width="6" height="8" rx="1"></rect>
   <line x1="14" y1="12" x2="20" y2="12"></line>
   <line x1="14" y1="16" x2="20" y2="16"></line>
   <line x1="14" y1="20" x2="20" y2="20"></line>
</svg>',
]);

menu()->add(101,[
	'url' => '/admin/plugins',
	'name' => '管理',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-package" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
    <line x1="12" y1="12" x2="20" y2="7.5"></line>
    <line x1="12" y1="12" x2="12" y2="21"></line>
    <line x1="12" y1="12" x2="4" y2="7.5"></line>
    <line x1="16" y1="5.25" x2="8" y2="9.75"></line>
 </svg>',
	'parent_id' => 1,
]);

menu()->add(102,[
	'url' => '/admin/plugins/upload',
	'name' => '上传',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-package" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
    <line x1="12" y1="12" x2="20" y2="7.5"></line>
    <line x1="12" y1="12" x2="12" y2="21"></line>
    <line x1="12" y1="12" x2="4" y2="7.5"></line>
    <line x1="16" y1="5.25" x2="8" y2="9.75"></line>
 </svg>',
	'parent_id' => 1,
]);
menu()->add(161,[
	'url' => '/admin/themes',
	'name' => '管理',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-package" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
    <line x1="12" y1="12" x2="20" y2="7.5"></line>
    <line x1="12" y1="12" x2="12" y2="21"></line>
    <line x1="12" y1="12" x2="4" y2="7.5"></line>
    <line x1="16" y1="5.25" x2="8" y2="9.75"></line>
 </svg>',
	'parent_id' => 6,
]);

menu()->add(162,[
	'url' => '/admin/themes/upload',
	'name' => '上传',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-package" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
    <line x1="12" y1="12" x2="20" y2="7.5"></line>
    <line x1="12" y1="12" x2="12" y2="21"></line>
    <line x1="12" y1="12" x2="4" y2="7.5"></line>
    <line x1="16" y1="5.25" x2="8" y2="9.75"></line>
 </svg>',
	'parent_id' => 6,
]);

// 网站设置
menu()->add(5,[
    'url' => '/admin/setting',
    'name' => '设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
   <circle cx="12" cy="12" r="3"></circle>
</svg>',
]);

menu()->add(10,[
   'url' => '#',
   'name' => '文件编辑',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-file" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M14 3v4a1 1 0 0 0 1 1h4"></path>
   <path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z"></path>
</svg>'
]);

menu()->add(1001,[
    'url' => '/admin/EditFile/css',
    'name' => '自定义CSS代码',
    'icon' => '',
    'parent_id' => 10
]);

menu()->add(1002,[
    'url' => '/admin/EditFile/js',
    'name' => '自定义JS代码',
    'icon' => '',
    'parent_id' => 10
]);
