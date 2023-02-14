<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
return [
    'handler' => Hyperf\Session\Handler\RedisHandler::class,
    'options' => [
        'connection' => 'default',
        'path' => BASE_PATH . '/runtime/session',
        'gc_maxlifetime' => 72 * 60 * 60,
        'session_name' => env('APP_KEY', 'CODEFEC_SESSION_ID'),
        'domain' => null,
        'cookie_lifetime' => 72 * 60 * 60,
    ],
];
