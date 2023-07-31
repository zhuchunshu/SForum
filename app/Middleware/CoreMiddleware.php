<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Middleware;

use FastRoute\Dispatcher;
use Hyperf\HttpServer\Router\Dispatched;
use Hyperf\Server\Exception\ServerException;
use Hyperf\Context\Context;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;

class CoreMiddleware extends \Hyperf\HttpServer\CoreMiddleware
{
    /**
     * Process an incoming server request and return a response, optionally delegating
     * response creation to a handler.
     */
    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        $request = Context::set(ServerRequestInterface::class, $request);

        /** @var Dispatched $dispatched */
        $dispatched = $request->getAttribute(Dispatched::class);

        if (! $dispatched instanceof Dispatched) {
            throw new ServerException(sprintf('The dispatched object is not a %s object.', Dispatched::class));
        }

        $response = null;
        switch ($dispatched->status) {
            case Dispatcher::NOT_FOUND:
                $response = $this->handleNotFound($request);
                break;
            case Dispatcher::METHOD_NOT_ALLOWED:
                $response = $this->handleMethodNotAllowed($dispatched->params, $request);
                break;
            case Dispatcher::FOUND:
                $response = $this->handleFound($dispatched, $request);
                break;
        }
        if (! $response instanceof ResponseInterface) {
            $response = $this->transferToResponse($response, $request);
        }
        return $response->withAddedHeader('Server', 'SForum');
    }

    /**
     * Handle the response when cannot found any routes.
     */
    protected function handleNotFound(ServerRequestInterface $request): mixed
    {
        // 重写路由找不到的处理逻辑
        return admin_abort(['msg' => '页面不存在'], 404);
    }

    /**
     * Handle the response when the routes found but doesn't match any available methods.
     */
    protected function handleMethodNotAllowed(array $methods, ServerRequestInterface $request): ResponseInterface
    {
        // 重写 HTTP 方法不允许的处理逻辑
        return $this->response()->withStatus(405);
    }
}
