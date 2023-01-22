<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */

namespace App\Plugins\User\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\User\src\Service\Middleware\Oauth2Master;
use App\Plugins\User\src\Service\Oauth2;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix: '/admin/setting/oauth2')]
#[Middleware(AdminMiddleware::class)]
class Oauth2Controller
{
    #[GetMapping(path: '')]
    public function index()
    {
        return view('User::Admin.oauth2.index');
    }

    #[PostMapping(path: '')]
    public function submit()
    {
        $request = request()->all();
        $handler = function ($request) {
            return redirect()->back()->with('success','更新成功!')->go();
        };

        // 通过中间件
        $run = $this->throughMiddleware($handler, $this->middlewares());
        return $run($request);
    }

    /**
     * 通过中间件 through the middleware.
     * @param $handler
     * @param $stack
     * @return \Closure|mixed
     */
    protected function throughMiddleware($handler, $stack): mixed
    {
        // 闭包实现中间件功能 closures implement middleware functions
        foreach ($stack as $middleware) {
            $handler = function ($request) use ($handler, $middleware) {
                if ($middleware instanceof \Closure) {
                    return call_user_func($middleware, $request, $handler);
                }

                return call_user_func([new $middleware(), 'handler'], $request, $handler);
            };
        }
        return $handler;
    }

    private function middlewares(): array
    {
        $_[] = Oauth2Master::class;
        $middlewares = array_merge($_, (new Oauth2())->get_all_handler());
        return array_reverse($middlewares);
    }
}
