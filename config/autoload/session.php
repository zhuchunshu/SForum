<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
use Hyperf\Session\Handler;

return [
    'handler' => Hyperf\Session\Handler\RedisHandler::class,
    'options' => [
        'connection' => 'default',
        'path' => BASE_PATH . '/runtime/session',
        'gc_maxlifetime' => 72 * 60 * 60,
        'session_name' => env("APP_NAME","CODEFEC_SESSION_ID"),
        'domain' => null,
        'cookie_lifetime' => 72 * 60 * 60,
    ],
];
