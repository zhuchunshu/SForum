<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Middleware;

use Hyperf\HttpServer\Router\Dispatched;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

class RouteRefuseMiddleware implements MiddlewareInterface
{
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
        if (! @$dispatched->handler->callback) {
            return $handler->handle($request);
        }
        if (! is_array($dispatched->handler->callback)) {
            return $handler->handle($request);
        }
        $Plugin = $dispatched->handler->callback[0];
        if (! $Plugin) {
            return $handler->handle($request);
        }
        $Plugin = explode('\\', $Plugin);
        if (count($Plugin) >= 1 && $Plugin[1] === 'Themes') {
            return response()->json(Json_Api(500, false, '禁止在主题内定义路由'));
        }
        if (count($Plugin) >= 3) {
            $Plugin = $Plugin[2];
        } else {
            return $handler->handle($request);
        }
        if (is_dir(BASE_PATH . '/app/Plugins/' . $Plugin) && ! in_array($Plugin, getEnPlugins(), true)) {
            return response()->json(Json_Api(500, false, '定义此路由的插件未启用'));
        }

        return $handler->handle($request);
    }
}
