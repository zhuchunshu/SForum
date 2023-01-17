<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */

namespace App\Plugins\User\src\Helpers;

use App\Plugins\User\src\Annotation\UserNameRenderAnnotation;
use Hyperf\Di\Annotation\AnnotationCollector;

class UserNameRender
{
    public function render($user, array $data)
    {
        @$user->usernameRender = $user->username;
        $handler = static function ($user, $data) {
            return @$user->usernameRender ?: $user->username;
        };

        // 通过中间件
        $run = $this->throughMiddleware($handler, $this->middlewares());
        return $run($user, $data);
    }

    /**
     * 通过中间件 through the middleware.
     * @param $handler
     * @param $stack
     * @return \Closure|mixed
     */
    protected function throughMiddleware($handler, $stack)
    {
        // 闭包实现中间件功能 closures implement middleware functions
        foreach ($stack as $middleware) {
            $handler = static function ($user, $data) use ($handler, $middleware) {
                if ($middleware instanceof \Closure) {
                    return $middleware($user, $data, $handler);
                }

                return call_user_func([new $middleware(), 'handler'], $user, $data, $handler);
            };
        }
        return $handler;
    }

    private function middlewares(): array
    {
        $middlewares = [];
        foreach (AnnotationCollector::getClassesByAnnotation(UserNameRenderAnnotation::class) as $key => $value) {
            $middlewares[] = $key;
        }
        return array_reverse($middlewares);
    }
}
