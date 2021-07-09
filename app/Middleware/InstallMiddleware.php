<?php

declare(strict_types=1);

namespace App\Middleware;

use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;

class InstallMiddleware implements MiddlewareInterface
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
        if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.lock")){
            if(request()->path()!="install"){
                return response()->redirect("/install");
            }
        }
        if(request()->path()=="install"){
            if(file_exists(BASE_PATH."/app/CodeFec/storage/install.lock")){
                return admin_abort(['msg' => '页面不存在'],404);
            }
        }
        return $handler->handle($request);
    }
}