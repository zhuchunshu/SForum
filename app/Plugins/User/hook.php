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

Itf()->add('user-admin-hook-credit', 1, [
    'name' => get_options('wealth_credit_name', '积分'),
    'view' => 'User::Admin.hook.credit.credit',
]);

Itf()->add('user-admin-hook-credit', 2, [
    'name' => get_options('wealth_golds_name', '金币'),
    'view' => 'User::Admin.hook.credit.gold',
]);

Itf()->add('user-admin-hook-credit', 3, [
    'name' => get_options('wealth_money_name', '余额'),
    'view' => 'User::Admin.hook.credit.money',
]);
