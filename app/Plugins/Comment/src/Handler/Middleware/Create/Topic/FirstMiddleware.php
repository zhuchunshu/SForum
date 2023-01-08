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

use App\Plugins\Comment\src\Annotation\Topic\CreateFirstMiddleware;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[CreateFirstMiddleware]
class FirstMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        if (! auth()->check()) {
            return redirect()->back()->with('danger', '未登陆')->go();
        }
        if (! Authority()->check('comment_create')) {
            return redirect()->back()->with('danger', '无评论权限')->go();
        }
        if (cache()->has('comment_create_time_' . auth()->id())) {
            $time = cache()->get('comment_create_time_' . auth()->id()) - time();
            return redirect()->back()->with('danger', '发表评论过于频繁,请 ' . $time . ' 秒后再试')->go();
        }
        cache()->set('comment_create_time_' . auth()->id(), time() + get_options('comment_create_time', 60), get_options('comment_create_time', 60));
        return $next($data);
    }
}
