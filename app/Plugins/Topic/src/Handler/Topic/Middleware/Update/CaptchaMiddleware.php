<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Update;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
#[\App\Plugins\Topic\src\Annotation\Topic\UpdateFirstMiddleware]
class CaptchaMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        //        if (! captcha()->check(request()->input('captcha'))) {
        //            return redirect()->back()->with('danger', '验证码错误')->go();
        //        }
        return $next($data);
    }
}