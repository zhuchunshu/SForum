<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
if (! function_exists('get_hook_credit_options')) {
    function get_hook_credit_options($name, $default = '')
    {
        return get_options('hook_user_credit_' . $name, $default);
    }
}

Itf()->add('users_home_menu', 12, [
    'name' => '任务',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-subtask" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M6 9l6 0"></path>
   <path d="M4 5l4 0"></path>
   <path d="M6 5v11a1 1 0 0 0 1 1h5"></path>
   <path d="M12 7m0 1a1 1 0 0 1 1 -1h6a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-6a1 1 0 0 1 -1 -1z"></path>
   <path d="M12 15m0 1a1 1 0 0 1 1 -1h6a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-6a1 1 0 0 1 -1 -1z"></path>
</svg>',
    'view' => 'User::assets.task',
    'quanxian' => \Opis\Closure\serialize((function ($user) {
        return (int) $user->id === auth()->id();
    })),
]);

Itf()->add('user-admin-hook-credit', 0, [
    'name' => '总开关',
    'view' => 'User::Admin.hook.credit.checkbox',
]);

Itf()->add('user-admin-hook-credit', 1, [
    'name' => get_options('wealth_credit_name', '积分'),
    'view' => 'User::Admin.hook.credit.credit',
]);

Itf()->add('user-admin-hook-credit', 2, [
    'name' => get_options('wealth_golds_name', '金币'),
    'view' => 'User::Admin.hook.credit.gold',
]);

//Itf()->add('user-admin-hook-credit', 3, [
//    'name' => get_options('wealth_money_name', '余额'),
//    'view' => 'User::Admin.hook.credit.money',
//]);

Itf()->add('user-admin-hook-credit', 4, [
    'name' => '设置头像',
    'view' => 'User::Admin.hook.credit.avatar',
]);

// 每日任务
Itf()->add('user_task_daily', 1, [
    'name' => '每日签到',
    'view' => 'User::assets.task.daily.checkin',
    'show' => function () {
        return get_hook_credit_options('checkin_check', 'true') === 'true';
    },
]);

Itf()->add('user_task_daily', 2, [
    'name' => '每日发帖',
    'view' => 'User::assets.task.daily.create_topic',
    'show' => function () {
        return get_hook_credit_options('create_topic_check', 'true') === 'true';
    },
]);

Itf()->add('user_task_daily', 4, [
    'name' => '每日评论',
    'view' => 'User::assets.task.daily.create_topic_comment',
    'show' =>function () {
        return get_hook_credit_options('create_topic_comment_check', 'true') === 'true';
    },
]);

Itf()->add('user_task_system', 1, [
    'name' => '设置头像',
    'view' => 'User::assets.task.system.set_avatar',
    'show' =>function () {
        return get_hook_credit_options('set_avatar_check', 'true') === 'true';
    },
]);
