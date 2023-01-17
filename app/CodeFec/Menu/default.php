<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
menu()->add(0, [
    'url' => '/admin',
    'name' => '仪表盘',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-dashboard" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
    <circle cx="12" cy="13" r="2"></circle>
    <line x1="13.45" y1="11.55" x2="15.5" y2="9.5"></line>
    <path d="M6.4 20a9 9 0 1 1 11.2 0z"></path>
 </svg>',
]);

menu()->add(1, [
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
menu()->add(6, [
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

menu()->add(101, [
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

menu()->add(102, [
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
menu()->add(161, [
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

menu()->add(162, [
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
menu()->add(5, [
    'url' => '/admin/setting',
    'name' => '站务',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-3d-cube-sphere" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M6 17.6l-2 -1.1v-2.5"></path>
   <path d="M4 10v-2.5l2 -1.1"></path>
   <path d="M10 4.1l2 -1.1l2 1.1"></path>
   <path d="M18 6.4l2 1.1v2.5"></path>
   <path d="M20 14v2.5l-2 1.12"></path>
   <path d="M14 19.9l-2 1.1l-2 -1.1"></path>
   <line x1="12" y1="12" x2="14" y2="10.9"></line>
   <line x1="18" y1="8.6" x2="20" y2="7.5"></line>
   <line x1="12" y1="12" x2="12" y2="14.5"></line>
   <line x1="12" y1="18.5" x2="12" y2="21"></line>
   <path d="M12 12l-2 -1.12"></path>
   <line x1="6" y1="8.6" x2="4" y2="7.5"></line>
</svg>',
]);



menu()->add(501, [
    'url' => '/admin/setting',
    'name' => '站点设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
   <circle cx="12" cy="12" r="3"></circle>
</svg>',
    'parent_id' => 5,
]);


menu()->add(504, [
    'url' => '/admin/setting/menu',
    'name' => '页头导航',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-category" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 4h6v6h-6z"></path>
   <path d="M14 4h6v6h-6z"></path>
   <path d="M4 14h6v6h-6z"></path>
   <circle cx="17" cy="17" r="3"></circle>
</svg>',
    'parent_id' => 5,
]);
menu()->add(522, [
    'url' => '/admin/server/backup',
    'name' => '备份',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-packge-export" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M12 21l-8 -4.5v-9l8 -4.5l8 4.5v4.5"></path>
   <path d="M12 12l8 -4.5"></path>
   <path d="M12 12v9"></path>
   <path d="M12 12l-8 -4.5"></path>
   <path d="M15 18h7"></path>
   <path d="M19 15l3 3l-3 3"></path>
</svg>',
    'parent_id' => 5,
]);


menu()->add(17,[
    'url' => '/admin/hook',
    'name' => '钩子',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-fish-hook" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16 9v6a5 5 0 0 1 -10 0v-4l3 3"></path>
   <circle cx="16" cy="7" r="2"></circle>
   <path d="M16 5v-2"></path>
</svg>',
]);

menu()->add(1712,[
    'url' => '/admin/hook/components',
    'name' => '部件',
    'icon' => '',
    'parent_id' => 17
]);