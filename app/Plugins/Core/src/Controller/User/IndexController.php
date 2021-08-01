<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\User\src\Middleware\LoginMiddleware;
use Exception;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use HyperfExt\Mail\Mail;
use Psr\Http\Message\ResponseInterface;

#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    /**
     * @throws Exception
     */
    #[GetMapping(path: "/test")]
    public function test()
    {
        return request()->fullUrl();
    }

    /**
     * 强制验证邮箱
     */
    #[GetMapping(path: "/user/ver_email")]
    public function user_ver_email(): ResponseInterface
    {
        if(auth()->data()->email_ver_time){
            return redirect()->url("/")->with("info","你已验证邮箱,无需重复操作")->go;
        }
        return view("plugins.Core.user.ver_email");
    }
}