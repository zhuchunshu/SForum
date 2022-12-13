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
    'create' => env('rate_limit_create', 1),
    'consume' => 1,
    'capacity' => env('rate_limit_capacity', 3),
    'limitCallback' => [],
    'waitTimeout' => 1,
];
