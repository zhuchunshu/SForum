<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 *
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
return [
    'create' => env("rate_limit_create",1),
    'consume' => 1,
    'capacity' => env("rate_limit_capacity",3),
    'limitCallback' => [],
    'waitTimeout' => 1,
];
