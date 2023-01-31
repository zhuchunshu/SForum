<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\Handler;

use App\Plugins\Mail\src\SendServiceHandlerInterface;

class Smtp implements SendServiceHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        if (arr_has($data, 'SMTP') && is_array($data['SMTP'])) {
            foreach ($data['SMTP'] as $key => $value) {
                set_options($key, $value);
            }
        }
        return $next($data);
    }
}
