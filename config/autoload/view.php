<?php

declare(strict_types=1);

use App\View\Component\CsrfToken;

/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
return [
    'engine' =>\App\CodeFec\View\HyperfViewEngine::class,
    'mode' => Hyperf\View\Mode::SYNC,
    'config' => [
        'view_path' => BASE_PATH . '/resources/views/',
        'cache_path' => BASE_PATH . '/runtime/view/',
    ],
    'components' => [
        'csrf' => CsrfToken::class,
    ],
];
