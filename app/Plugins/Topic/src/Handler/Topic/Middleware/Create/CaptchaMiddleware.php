<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[\App\Plugins\Topic\src\Annotation\Topic\CreateFirstMiddleware]
class CaptchaMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
//        if (! captcha()->check(request()->input('captcha'))) {
//            unset($data['basis']['content']);
//            return redirect()->with('danger', '验证码错误')->url('topic/create?' . http_build_query($data))->go();
//        }
        return $next($data);
    }
}
