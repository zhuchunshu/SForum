<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
Itf_Setting()->add(
    200,
    '主题 - 全局',
    'theme-common',
    'setting.theme.common'
);

Itf_Setting()->add(
    201,
    '主题 - 页头',
    'theme-header',
    'setting.theme.header'
);
Itf()->add('core_menu', 1, [
    'name' => '首页',
    'url' => '/',
]);

Itf_Setting()->add(
    212,
    '登陆注册',
    'setting_user_sign',
    'setting.user.sign'
);
Itf_Setting()->add(
    204,
    '用户设置',
    'user-setting',
    'setting.user.core'
);
Itf_Setting()->add(
    205,
    '短信设置',
    'user-sms',
    'setting.user.sms'
);

Itf()->add('ui-topic-page-right-layout', 1, [
    'view' => 'App::topic.right',
    'enable' => (function () {
        return true;
    }),
]);

menu()->add(2001, [
    'name' => '注册邀请码',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
    'url' => '/admin/Invitation-code',
]);

menu()->add(2002, [
    'name' => '管理',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
    'url' => '/admin/Invitation-code',
    'parent_id' => 2001,
]);

menu()->add(2003, [
    'name' => '添加',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
    'url' => '/admin/Invitation-code/create',
    'parent_id' => 2001,
]);

menu()->add(2004, [
    'name' => '导出',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-key" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8" cy="15" r="4"></circle>
   <line x1="10.85" y1="12.15" x2="19" y2="4"></line>
   <line x1="18" y1="5" x2="20" y2="7"></line>
   <line x1="15" y1="8" x2="17" y2="10"></line>
</svg>',
    'url' => '/admin/Invitation-code/export',
    'parent_id' => 2001,
]);

Itf()->add('SMS', 1, [
    'name' => 'Qcloud',
    'handler' => \App\Plugins\Core\src\Lib\Sms\Service\Qcloud::class,
    'view' => 'App::Sms.qcloud',
]);

Itf()->add('SMS', 2, [
    'name' => 'Ucloud',
    'handler' => \App\Plugins\Core\src\Lib\Sms\Service\Ucloud::class,
    'view' => 'App::Sms.ucloud',
]);

Itf()->add('SMS', 3, [
    'name' => 'SmsBao',
    'handler' => \App\Plugins\Core\src\Lib\Sms\Service\SmsBao::class,
    'view' => 'App::Sms.smsbao',
]);

menu()->add(71, [
    'name' => '支付',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-brand-paypal" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10 13l2.5 0c2.5 0 5 -2.5 5 -5c0 -3 -1.9 -5 -5 -5h-5.5c-.5 0 -1 .5 -1 1l-2 14c0 .5 .5 1 1 1h2.8l1.2 -5c.1 -.6 .4 -1 1 -1zm7.5 -5.8c1.7 1 2.5 2.8 2.5 4.8c0 2.5 -2.5 4.5 -5 4.5h-2.6l-.6 3.6a1 1 0 0 1 -1 .8l-2.7 0a0.5 .5 0 0 1 -.5 -.6l.2 -1.4"></path>
</svg>',
    'url' => '/admin/Pay',
]);
menu()->add(3702, [
    'name' => '配置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-brand-paypal" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10 13l2.5 0c2.5 0 5 -2.5 5 -5c0 -3 -1.9 -5 -5 -5h-5.5c-.5 0 -1 .5 -1 1l-2 14c0 .5 .5 1 1 1h2.8l1.2 -5c.1 -.6 .4 -1 1 -1zm7.5 -5.8c1.7 1 2.5 2.8 2.5 4.8c0 2.5 -2.5 4.5 -5 4.5h-2.6l-.6 3.6a1 1 0 0 1 -1 .8l-2.7 0a0.5 .5 0 0 1 -.5 -.6l.2 -1.4"></path>
</svg>',
    'url' => '/admin/Pay/config',
    'parent_id' => 71,
]);
menu()->add(3703, [
    'name' => '设置',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-brand-paypal" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10 13l2.5 0c2.5 0 5 -2.5 5 -5c0 -3 -1.9 -5 -5 -5h-5.5c-.5 0 -1 .5 -1 1l-2 14c0 .5 .5 1 1 1h2.8l1.2 -5c.1 -.6 .4 -1 1 -1zm7.5 -5.8c1.7 1 2.5 2.8 2.5 4.8c0 2.5 -2.5 4.5 -5 4.5h-2.6l-.6 3.6a1 1 0 0 1 -1 .8l-2.7 0a0.5 .5 0 0 1 -.5 -.6l.2 -1.4"></path>
</svg>',
    'url' => '/admin/Pay/setting',
    'parent_id' => 71,
]);
menu()->add(3701, [
    'name' => '订单',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-brand-paypal" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10 13l2.5 0c2.5 0 5 -2.5 5 -5c0 -3 -1.9 -5 -5 -5h-5.5c-.5 0 -1 .5 -1 1l-2 14c0 .5 .5 1 1 1h2.8l1.2 -5c.1 -.6 .4 -1 1 -1zm7.5 -5.8c1.7 1 2.5 2.8 2.5 4.8c0 2.5 -2.5 4.5 -5 4.5h-2.6l-.6 3.6a1 1 0 0 1 -1 .8l-2.7 0a0.5 .5 0 0 1 -.5 -.6l.2 -1.4"></path>
</svg>',
    'url' => '/admin/Pay',
    'parent_id' => 71,
]);

Itf()->add('Pay', 1, [
    'name' => '微信支付',
    'ename' => 'wechatPay',
    'description' => '【官方】微信支付',
    'handler' => \App\Plugins\Core\src\Lib\Pay\Service\WechatPay::class,
    'icon' => '/plugins/Core/image/wechatPay_icon.svg',
    'logo' => '/plugins/Core/image/wechatPay.png',
    'view' => 'App::Pay.wechatPay',
]);

Itf()->add('Pay', 2, [
    'name' => '支付宝',
    'ename' => 'aliPay',
    'description' => '【官方】支付宝',
    'handler' => \App\Plugins\Core\src\Lib\Pay\Service\AliPay::class,
    'icon' => '/plugins/Core/image/alipay_icon.svg',
    'logo' => '/plugins/Core/image/alipay.svg',
    'view' => 'App::Pay.aliPay',
]);

Itf()->add('Pay', 0, [
    'name' => '余额支付',
    'ename' => 'SFPay',
    'description' => '优先使用账户余额支付',
    'handler' => \App\Plugins\Core\src\Lib\Pay\Service\SFPay::class,
    'icon' => '/plugins/Core/image/PAYMENT.svg',
    'logo' => '/plugins/Core/image/PAYMENT.svg',
    'view' => 'App::Pay.SFPay',
]);

menu()->add(511, [
    'url' => '/admin/setting/friend_links',
    'name' => '友情链接',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-link" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M10 14a3.5 3.5 0 0 0 5 0l4 -4a3.5 3.5 0 0 0 -5 -5l-.5 .5"></path>
   <path d="M14 10a3.5 3.5 0 0 0 -5 0l-4 4a3.5 3.5 0 0 0 5 5l.5 -.5"></path>
</svg>',
    'parent_id' => 5,
]);
