<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Create\Topic;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

class EndMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        return redirect()->back()->with('success', '发表成功!')->go();
    }
}
