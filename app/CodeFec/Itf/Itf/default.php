<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
Itf()->add('csrf', 1, 'csrf_test\\/*');
Itf()->add('csrf', 2, 'api/pay/*');
Itf()->add('error_redirect', 1, [
    '/admin' => '/admin/login',
    '/admin/login' => '/admin',
]);
