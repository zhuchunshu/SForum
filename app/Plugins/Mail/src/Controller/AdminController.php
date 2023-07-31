<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\Controller;

use App\Middleware\AdminMiddleware;
use App\Plugins\Mail\src\Service\MailMaster;
use App\Plugins\Mail\src\Service\SendService;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Middleware(AdminMiddleware::class)]
#[Controller(prefix: '/admin/mail')]
class AdminController
{
    #[GetMapping('')]
    public function index()
    {
        return view('Mail::admin.index');
    }
    #[PostMapping('')]
    public function submit()
    {
        $request = request()->all();
        $handler = function ($request) {
            return redirect()->back()->with('success', '更新成功!')->go();
        };
        // 通过中间件
        $run = $this->throughMiddleware($handler, $this->middlewares());
        return $run($request);
    }
    #[GetMapping('test')]
    public function test()
    {
        return view('Mail::admin.test');
    }
    #[PostMapping('test')]
    public function test_submit()
    {
        $email = request()->input('email');
        if (!$email) {
            return redirect()->back()->with('danger', '请求参数不足!')->go();
        }
        $Subject = '【' . get_options('web_name') . '】 邮件测试!';
        $Body = <<<'HTML'
当你收到此条消息,说明邮件发送成功!
HTML;
        if (Email()->send($email, $Subject, $Body)) {
            return redirect()->back()->with('success', '发送成功!')->go();
        }
        return redirect()->back()->with('danger', '发送失败!')->go();
    }
    /**
     * 通过中间件 through the middleware.
     * @param $handler
     * @param $stack
     * @return \Closure|mixed
     */
    protected function throughMiddleware($handler, $stack) : mixed
    {
        // 闭包实现中间件功能 closures implement middleware functions
        foreach ($stack as $middleware) {
            $handler = function ($request) use($handler, $middleware) {
                if ($middleware instanceof \Closure) {
                    return call_user_func($middleware, $request, $handler);
                }
                return call_user_func([new $middleware(), 'handler'], $request, $handler);
            };
        }
        return $handler;
    }
    private function middlewares() : array
    {
        $_[] = MailMaster::class;
        $middlewares = array_merge($_, (new SendService())->get_handlers());
        return array_reverse($middlewares);
    }
}