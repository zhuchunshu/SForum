<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Middleware;

use Hyperf\Utils\Str;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;
/**
 * Auth 组件的基本验证
 */
class AuthMiddleware implements MiddlewareInterface
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
    }
    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler) : ResponseInterface
    {
        if (auth()->check()) {
            $auth = auth()->data();
            if (Str::is('register', request()->path()) || Str::is('login*', request()->path())) {
                return admin_abort(['msg' => '您已登录']);
            }
            foreach (Itf()->get('authMiddleware') as $value) {
                if (Str::is($value, request()->path())) {
                    return $handler->handle($request);
                }
            }
            // 邮箱验证
            if ((int) get_options('core_user_email_ver', 1) === 1 && !strtotime(@$auth->email_ver_time ?: '1') && request()->path() !== 'user/ver_email' && request()->path() !== 'user/ver_phone' && request()->path() !== 'user/ver_phone/send') {
                return redirect()->url('/user/ver_email')->go();
            }
        }
        return $handler->handle($request);
    }
}