<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Plugins\User\src\Models\UsersNotice;

Itf()->add('authMiddleware', 1, 'api*');
Itf()->add('authMiddleware', 2, 'admin*');
Itf()->add('authMiddleware', 3, 'logout');

// 用户设置
//Itf()->add("users_options",1,[
//	'name' => '1',
//	'view' => '2'
//]);
Itf()->add('userSetting', 1, [
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
    'view' => 'App::user.setting.common',
]);

Itf()->add('userSetting', 2, [
    'name' => '灵活设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/settings</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
   <circle cx="12" cy="12" r="3"></circle>
</svg>',
    'view' => 'App::user.setting.options',
]);

Itf()->add('userSetting', 3, [
    'name' => '背景图',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-photo" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="15" y1="8" x2="15.01" y2="8"></line>
   <rect x="4" y="4" width="16" height="16" rx="3"></rect>
   <path d="M4 15l4 -4a3 5 0 0 1 3 0l5 5"></path>
   <path d="M14 14l1 -1a3 5 0 0 1 3 0l2 2"></path>
</svg>',
    'view' => 'App::user.setting.backgroundImg',
]);

Itf()->add('users_settings', 1, [
    'name' => '自定义代码',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/settings</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
   <circle cx="12" cy="12" r="3"></circle>
</svg>',
    'view' => 'User::setting.user.code',
]);

Itf()->add('users_settings', 2, [
    'name' => '私信设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-messages" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/messages</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M21 14l-3 -3h-7a1 1 0 0 1 -1 -1v-6a1 1 0 0 1 1 -1h9a1 1 0 0 1 1 1v10"></path>
   <path d="M14 15v2a1 1 0 0 1 -1 1h-7l-3 3v-10a1 1 0 0 1 1 -1h2"></path>
</svg>',
    'view' => 'User::setting.user.message',
]);

Itf_Setting()->add(
    171,
    '财富设置',
    'wealth',
    'User::setting.admin.wealth'
);

Itf_Setting()->add(
    172,
    '私信设置',
    'pm',
    'User::setting.admin.pm'
);

Itf()->add('users_home_menu', 1, [
    'name' => '概览',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/user</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="7" r="4"></circle>
   <path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
</svg>',
    'view' => 'User::home.overview',
    'quanxian' => null,
]);

Itf()->add('users_home_menu', 2, [
    'name' => '主题',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-note" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/note</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="13" y1="20" x2="20" y2="13"></line>
   <path d="M13 20v-6a1 1 0 0 1 1 -1h6v-7a2 2 0 0 0 -2 -2h-12a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7"></path>
</svg>',
    'view' => 'User::home.topic',
    'quanxian' => null,
]);

Itf()->add('users_home_menu', 3, [
    'name' => '评论',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message-dots" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/message-dots</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 21v-13a3 3 0 0 1 3 -3h10a3 3 0 0 1 3 3v6a3 3 0 0 1 -3 3h-9l-4 4"></path>
   <line x1="12" y1="11" x2="12" y2="11.01"></line>
   <line x1="8" y1="11" x2="8" y2="11.01"></line>
   <line x1="16" y1="11" x2="16" y2="11.01"></line>
</svg>',
    'view' => 'User::home.comment',
    'quanxian' => null,
]);

Itf()->add('users_home_menu', 4, [
    'name' => '关注',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/users</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>',
    'view' => 'User::home.following',
    'quanxian' => null,
]);

Itf()->add('users_home_menu', 5, [
    'name' => '粉丝',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/users</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>',
    'view' => 'User::home.fans',
    'quanxian' => null,
]);

Itf()->add('users_home_menu', 6, [
    'name' => '收藏',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-star" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/star</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
</svg>',
    'view' => 'User::home.collections',
    'quanxian' => null,
]);
Itf()->add('users_home_menu', 7, [
    'name' => '订单',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-credit-card" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <rect x="3" y="5" width="18" height="14" rx="3"></rect>
   <line x1="3" y1="10" x2="21" y2="10"></line>
   <line x1="7" y1="15" x2="7.01" y2="15"></line>
   <line x1="11" y1="15" x2="13" y2="15"></line>
</svg>',
    'view' => 'User::home.order',
    'quanxian' => \Opis\Closure\serialize((function ($user) {
        return (int) $user->id === auth()->id();
    })),
]);

Itf()->add('users_notices', 1, [
    'name' => '互动通知',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-arrows-right-left" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/arrows-right-left</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="21" y1="7" x2="3" y2="7"></line>
   <path d="M18 10l3 -3l-3 -3"></path>
   <path d="M6 20l-3 -3l3 -3"></path>
   <line x1="3" y1="17" x2="21" y2="17"></line>
</svg>',
    'view' => 'User::notice.interactive',
    'count' => \Opis\Closure\serialize(function ($user_id) {
        return UsersNotice::query()->where(['user_id' => $user_id, 'status' => 'publish'])->count();
    }),
]);

//Itf()->add('users_notices',2,[
//	'name' => '系统通知',
//	'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-bell-ringing" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
//   <desc>Download more icon variants from https://tabler-icons.io/i/bell-ringing</desc>
//   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
//   <path d="M10 5a2 2 0 0 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6"></path>
//   <path d="M9 17v1a3 3 0 0 0 6 0v-1"></path>
//   <path d="M21 6.727a11.05 11.05 0 0 0 -2.794 -3.727"></path>
//   <path d="M3 6.727a11.05 11.05 0 0 1 2.792 -3.727"></path>
//</svg>',
//	'view' => 'User::notice.system',
//]);

Itf()->add('users_notices', 3, [
    'name' => '私信通知',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-messages" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/messages</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M21 14l-3 -3h-7a1 1 0 0 1 -1 -1v-6a1 1 0 0 1 1 -1h9a1 1 0 0 1 1 1v10"></path>
   <path d="M14 15v2a1 1 0 0 1 -1 1h-7l-3 3v-10a1 1 0 0 1 1 -1h2"></path>
</svg>',
    'view' => 'User::notice.pm',
    'count' => \Opis\Closure\serialize(function ($user_id) {
        return \App\Plugins\User\src\Models\UsersPm::query()->where('to_id', $user_id)->where('read', false)->count();
    }),
]);

Itf()->add('users_home_menu', 8, [
    'name' => '充值',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-shopping-cart" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="6" cy="19" r="2"></circle>
   <circle cx="17" cy="19" r="2"></circle>
   <path d="M17 17h-11v-14h-2"></path>
   <path d="M6 5l14 1l-1 7h-13"></path>
</svg>',
    'view' => 'User::assets.money_recharge',
    'quanxian' => \Opis\Closure\serialize((function ($user) {
        return (int) $user->id === auth()->id();
    })),
]);

Itf()->add('users_home_menu', 9, [
    'name' => '兑换',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-exchange" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="5" cy="18" r="2"></circle>
   <circle cx="19" cy="6" r="2"></circle>
   <path d="M19 8v5a5 5 0 0 1 -5 5h-3l3 -3m0 6l-3 -3"></path>
   <path d="M5 16v-5a5 5 0 0 1 5 -5h3l-3 -3m0 6l3 -3"></path>
</svg>',
    'view' => 'User::assets.exchange',
    'quanxian' => \Opis\Closure\serialize((function ($user) {
        return (int) $user->id === auth()->id();
    })),
]);

Itf()->add('user_exchange', 0, [
    'name' => function () {
        return get_options('wealth_money_name', '余额') . ' >>> ' . get_options('wealth_golds_name', '金币');
    },
    'view' => 'User::assets.exchange.moneyTo_golds',
    'url' => '/api/user/exchange/moneyTo_golds',
    'on' => function () {
        return get_options('wealth_close_redemption_money_to_golds') !== 'true';
    },
]);

Itf()->add('user_exchange', 1, [
    'name' => function () {
        return get_options('wealth_money_name', '余额') . ' >>> ' . get_options('wealth_credit_name', '积分');
    },
    'view' => 'User::assets.exchange.moneyTo_credit',
    'url' => '/api/user/exchange/moneyTo_credit',
    'on' => function () {
        return get_options('wealth_close_redemption_money_to_credit') !== 'true';
    },
]);

Itf()->add('user_exchange', 2, [
    'name' => function () {
        return get_options('wealth_golds_name', '金币') . ' >>> ' . get_options('wealth_credit_name', '积分');
    },
    'view' => 'User::assets.exchange.goldsTo_credit',
    'url' => '/api/user/exchange/goldsTo_credit',
    'on' => function () {
        return get_options('wealth_close_redemption_golds_to_credit') !== 'true';
    },
]);


Itf()->add('userSetting', 10, [
    'name' => '登陆信息',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16.555 3.843l3.602 3.602a2.877 2.877 0 0 1 0 4.069l-2.643 2.643a2.877 2.877 0 0 1 -4.069 0l-.301 -.301l-6.558 6.558a2 2 0 0 1 -1.239 .578l-.175 .008h-1.172a1 1 0 0 1 -.993 -.883l-.007 -.117v-1.172a2 2 0 0 1 .467 -1.284l.119 -.13l.414 -.414h2v-2h2v-2l2.144 -2.144l-.301 -.301a2.877 2.877 0 0 1 0 -4.069l2.643 -2.643a2.877 2.877 0 0 1 4.069 0z"></path>
   <path d="M15 9h.01"></path>
</svg>',
    'view' => 'App::user.setting.auth',
]);