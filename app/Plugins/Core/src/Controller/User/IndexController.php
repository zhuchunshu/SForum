<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    #[GetMapping(path: "/user/setting")]
    public function user_setting(): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Core.user.setting");
    }
}