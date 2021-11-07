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

use Hyperf\ExceptionHandler\Handler\WhoopsExceptionHandler;

return [
    'handler' => [
        'http' => [
            Hyperf\HttpServer\Exception\Handler\HttpExceptionHandler::class,
            App\Exception\Handler\AppExceptionHandler::class,
            App\Exception\Handler\RateLimitExceptionHandler::class,
            App\Exception\Handler\ValidationExceptionHandler::class,
            WhoopsExceptionHandler::class,
        ],
    ],
];
