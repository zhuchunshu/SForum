<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Controller\AdminController;
use Hyperf\HttpServer\Router\Router;

// 后台登陆
Router::addRoute(['GET'], '/admin/login', [AdminController::class, 'login']);
Router::addRoute(['POST'], '/admin/login', [AdminController::class, 'loginPost']);

// 后台路由组
Router::addGroup('/admin', function () {
    Router::addRoute(['GET', 'POST'], '', [AdminController::class, 'index']);
    Router::post('/logout', [AdminController::class, 'logout']);
}, [
    'middleware' => [\App\Middleware\AdminMiddleware::class],
]);
