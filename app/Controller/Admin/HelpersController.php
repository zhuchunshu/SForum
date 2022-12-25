<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix: '/admin/helpers')]
#[Middleware(AdminMiddleware::class)]
#[Middleware(LoginMiddleware::class)]
class HelpersController
{
    #[GetMapping(path: 'clear_sessions')]
    public function clear_sessions(): \Psr\Http\Message\ResponseInterface
    {
        session()->clear();
        return redirect()->url('/')->with('success', 'Session 清理完毕')->go();
    }
}
