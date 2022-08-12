<?php
Itf_Setting()->add(200,
    "主题 - 全局","theme-common","setting.theme.common");

Itf_Setting()->add(201,
"主题 - 页头","theme-header","setting.theme.header");
Itf()->add("core_menu",1,[
    "name" => "首页",
    "url" => "/"
]);

Itf_Setting()->add(212,
    "登陆注册","setting_user_sign","setting.user.sign");
Itf_Setting()->add(204,
    "用户设置","user-setting","setting.user.core");

Itf()->add("show_right",1,"App::topic.right");

menu()->add(2001,[
	'name' => '注册邀请码',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
	'url' => '/admin/Invitation-code'
]);

menu()->add(2002,[
	'name' => '管理',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
	'url' => '/admin/Invitation-code',
	'parent_id' => 2001
]);

menu()->add(2003,[
	'name' => '添加',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
	'url' => '/admin/Invitation-code/create',
	'parent_id' => 2001
]);

menu()->add(2004,[
	'name' => '导出',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
	'url' => '/admin/Invitation-code/export',
	'parent_id' => 2001
]);