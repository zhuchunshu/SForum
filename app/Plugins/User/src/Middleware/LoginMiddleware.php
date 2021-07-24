<?php


namespace App\Plugins\User\src\Middleware;


use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

/**
 * 强制要求登陆
 * @package App\Plugins\User\src\Middleware
 */
class LoginMiddleware implements MiddlewareInterface
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
        if(!auth()->check()){
            if(request()->path() !== "register" && request()->path() !== "login"){
                return admin_abort(['msg' => '登录后才可访问','back' => '/login']);
            }
        }
        return $handler->handle($request);
    }
}