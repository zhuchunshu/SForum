<?php

declare(strict_types=1);

namespace App\Middleware;

use Csrf\Csrf;
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
        if(config("codefec.app.csrf")){
            if(request()->isMethod("post")){
                if(csrf_token()!=request()->input("_token")){
                    return admin_abort(["msg" => "会话超时,请重新提交"],419);
                }
            }
        }
        return $handler->handle($request);
    }
}