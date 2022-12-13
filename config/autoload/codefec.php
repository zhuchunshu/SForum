<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
return [
    'app' => [
        'name' => env('APP_KEY', 'CodeFec'),
        'csrf' => (bool) env('CODEFEC_APP_CSRF', true),
    ],
];
