<?php

// 设置邮件验证白名单路径
Itf()->add("authMiddleware",1,"api*");
Itf()->add("authMiddleware",2,"admin*");
Itf()->add("authMiddleware",3,"logout");

// 用户设置
//Itf()->add("users_options",1,[
//	'name' => '1',
//	'view' => '2'
//]);
Itf()->add('userSetting',1,[
	'name' => '基本设置',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-sm-block"
                                                 width="24"
                                                 height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                                 fill="none"
                                                 stroke-linecap="round" stroke-linejoin="round">
                                                <path d="M0 0h24v24H0z" stroke="none"/>
                                                <circle cx="12" cy="12" r="9"/>
                                                <path d="M12 7v5l3 3"/>
                                            </svg>',
	'view' => 'App::user.setting.common'
]);

Itf()->add('userSetting',2,[
	'name' => '灵活设置',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/settings</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
   <circle cx="12" cy="12" r="3"></circle>
</svg>',
	'view' => 'App::user.setting.options'
]);

Itf()->add('userSetting',3,[
	'name' => '背景图',
	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-photo" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="15" y1="8" x2="15.01" y2="8"></line>
   <rect x="4" y="4" width="16" height="16" rx="3"></rect>
   <path d="M4 15l4 -4a3 5 0 0 1 3 0l5 5"></path>
   <path d="M14 14l1 -1a3 5 0 0 1 3 0l2 2"></path>
</svg>',
	'view' => 'App::user.setting.backgroundImg'
]);
