<?php


namespace App\Plugins\User\src\Middleware;


use App\CodeFec\Admin\Admin;
use Hyperf\Utils\Str;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

/**
 * Auth 组件的基本验证
 * @package App\Plugins\User\src\Middleware
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

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        if(auth()->check()){
            if(request()->path() === "register" || request()->path() === "login"){
                return admin_abort(['msg' => '您已登录']);
            }
            foreach(Itf()->get("authMiddleware") as $value){
                if(Str::is($value,request()->path())){
                    return $handler->handle($request);
                }
            }
            if(!auth()->data()->email_ver_time && request()->path() !== "user/ver_email"){
                return redirect()->url("/user/ver_email")->go();
            }
        }
        return $handler->handle($request);
    }

}