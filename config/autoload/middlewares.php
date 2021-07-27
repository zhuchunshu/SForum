<?php

declare(strict_types=1);

use App\Middleware\CsrfMiddleware;
use App\Middleware\ShareErrorsFromSession;
use App\Middleware\ValidationExceptionHandle;

/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
return [
    'http' => [
        \Hyperf\Session\Middleware\SessionMiddleware::class,
        \Hyperf\Validation\Middleware\ValidationMiddleware::class,
        CsrfMiddleware::class,
        \App\Middleware\InstallMiddleware::class,
        \App\Middleware\RewriteMiddleware::class,

    ],
];
