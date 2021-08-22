<?php

declare(strict_types=1);

namespace App\Middleware;

use Hyperf\HttpServer\Router\Dispatched;
use Hyperf\Utils\Arr;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;

class RouteRefuseMiddleware implements MiddlewareInterface
{
    /**
     * @var ContainerInterface
     */
    protected ContainerInterface $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
    }

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        /** @var Dispatched $dispatched */
        $dispatched = $request->getAttribute(Dispatched::class);
        // 插件名称
        if(!@$dispatched->handler->callback){
            return $handler->handle($request);
        }
        if(!is_array($dispatched->handler->callback)){
            return $handler->handle($request);
        }
        $Plugin = $dispatched->handler->callback[0];
        if(!$Plugin){
            return $handler->handle($request);
        }
        $Plugin = explode('\\', $Plugin);
        if(count($Plugin)>=3){
            $Plugin = $Plugin[2];
        }else{
            return $handler->handle($request);
        }
        if(is_dir(BASE_PATH . "/app/Plugins/" . $Plugin) && !in_array($Plugin, Plugins_EnList(), true)) {
            return admin_abort($Plugin."插件未启用,无法访问此插件定义的路由",401);
        }

        return $handler->handle($request);
    }
}