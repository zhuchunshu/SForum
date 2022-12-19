<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[\App\Plugins\Topic\src\Annotation\Topic\CreateLastMiddleware]
class CreateLastMiddleware implements MiddlewareInterface
{
    public function handler($data,\Closure $next)
    {
        cache()->set('topic_create_time_' . auth()->id(), time() + get_options('topic_create_time', 120), get_options('topic_create_time', 120));
        return $next($data);
    }
}
