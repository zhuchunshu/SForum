<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Edit\Topic;

use App\Plugins\Comment\src\Annotation\Topic\UpdateLastMiddleware;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[UpdateLastMiddleware]
class LastMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        return $next($data);
    }
}
