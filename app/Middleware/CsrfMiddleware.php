<?php

declare(strict_types=1);

namespace App\Middleware;

use Csrf\Csrf;
use Hyperf\Utils\Str;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;

class CsrfMiddleware implements MiddlewareInterface
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
        if(!config("codefec.app.csrf")){
            return $handler->handle($request);
        }
        foreach(Itf()->get("csrf") as $value){
            if(Str::is($value,request()->path())){
                return $handler->handle($request);
            }
        }
        if(request()->isMethod("post") && csrf_token() !== request()->input("_token")) {
            return admin_abort(["msg" => "会话超时,请刷新后重新提交"],419);
        }

        return $handler->handle($request);
    }
}