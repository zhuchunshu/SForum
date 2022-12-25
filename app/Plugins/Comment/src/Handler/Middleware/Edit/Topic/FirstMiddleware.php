<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Edit\Topic;

use App\Plugins\Comment\src\Annotation\Topic\UpdateFirstMiddleware;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[UpdateFirstMiddleware]
class FirstMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        if (! auth()->check()) {
            return redirect()->back()->with('danger', '未登陆')->go();
        }
        return $next($data);
    }
}
