<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\View\Component\CsrfToken;

return [
    'engine' => \App\CodeFec\View\HyperfViewEngine::class,
    'mode' => Hyperf\View\Mode::SYNC,
    'config' => [
        'view_path' => BASE_PATH . '/resources/views/',
        'cache_path' => BASE_PATH . '/runtime/view/',
    ],
    'components' => [
        'csrf' => CsrfToken::class,
    ],
];
