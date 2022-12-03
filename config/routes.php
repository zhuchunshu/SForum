<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
use App\CodeFec\Plugins;
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

// 扩展路由

foreach ((new Plugins())->getEnPlugins() as $value) {
    if (file_exists(plugin_path($value . '/routes.php'))) {
        require plugin_path($value . '/routes.php');
    }
}
