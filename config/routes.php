<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */

use App\CodeFec\Plugins;
use App\Controller\AdminController;
use App\Controller\InstallController;
use Hyperf\HttpServer\Router\Router;



// 后台登陆
Router::addRoute(['GET'], '/admin/login', [AdminController::class,"login"]);
Router::addRoute(['POST'], '/admin/login', [AdminController::class,"loginPost"]);

// 后台路由组
Router::addGroup("/admin",function(){
    Router::addRoute(['GET','POST'],"",[AdminController::class,"index"]);
    Router::post("/logout",[AdminController::class,"logout"]);
},[
    "middleware" => [\App\Middleware\AdminMiddleware::class]
]);


// 扩展路由

foreach ((new Plugins())->getEnPlugins() as $value){
    if(file_exists(plugin_path($value."/routes.php"))){
        require plugin_path($value."/routes.php");
    }
}