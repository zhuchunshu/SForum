<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Server;

use FastRoute\Dispatcher;
use Hyperf\Contract\ConfigInterface;
use Hyperf\Contract\MiddlewareInitializerInterface;
use Hyperf\Contract\OnRequestInterface;
use Hyperf\Dispatcher\HttpDispatcher;
use Hyperf\ExceptionHandler\ExceptionHandlerDispatcher;
use Hyperf\HttpMessage\Server\Request as Psr7Request;
use Hyperf\HttpMessage\Server\Response as Psr7Response;
use Hyperf\HttpServer\Contract\CoreMiddlewareInterface;
use Hyperf\HttpServer\CoreMiddleware;
use Hyperf\HttpServer\Exception\Handler\HttpExceptionHandler;
use Hyperf\HttpServer\MiddlewareManager;
use Hyperf\HttpServer\ResponseEmitter;
use Hyperf\HttpServer\Router\Dispatched;
use Hyperf\HttpServer\Router\DispatcherFactory;
use Hyperf\Context\Context;
use Hyperf\Utils\Coordinator\Constants;
use Hyperf\Utils\Coordinator\CoordinatorManager;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Throwable;

class CodeFecServer implements OnRequestInterface, MiddlewareInitializerInterface
{
    protected ContainerInterface $container;

    protected HttpDispatcher $dispatcher;

    protected ExceptionHandlerDispatcher $exceptionHandlerDispatcher;

    protected array $middlewares;

    protected CoreMiddlewareInterface $coreMiddleware;

    protected array $exceptionHandlers;

    protected Dispatcher $routerDispatcher;

    protected ResponseEmitter $responseEmitter;

    protected string $serverName;

    public function __construct(ContainerInterface $container, HttpDispatcher $dispatcher, ExceptionHandlerDispatcher $exceptionHandlerDispatcher, ResponseEmitter $responseEmitter)
    {
        $this->container = $container;
        $this->dispatcher = $dispatcher;
        $this->exceptionHandlerDispatcher = $exceptionHandlerDispatcher;
        $this->responseEmitter = $responseEmitter;
    }

    public function initCoreMiddleware(string $serverName): void
    {
        $this->serverName = $serverName;
        $this->coreMiddleware = $this->createCoreMiddleware();
        $this->routerDispatcher = $this->createDispatcher($serverName);

        $config = $this->container->get(ConfigInterface::class);
        $this->middlewares = $config->get('middlewares.' . $serverName, []);
        $this->exceptionHandlers = $config->get('exceptions.handler.' . $serverName, $this->getDefaultExceptionHandler());
    }

    public function onRequest($request, $response): void
    {
        try {
            CoordinatorManager::until(Constants::WORKER_START)->yield();

            [$psr7Request, $psr7Response] = $this->initRequestAndResponse($request, $response);

            $psr7Request = $this->coreMiddleware->dispatch($psr7Request);
            /** @var Dispatched $dispatched */
            $dispatched = $psr7Request->getAttribute(Dispatched::class);
            $middlewares = $this->middlewares;
            if ($dispatched->isFound()) {
                $registeredMiddlewares = MiddlewareManager::get($this->serverName, $dispatched->handler->route, $psr7Request->getMethod());
                $middlewares = array_merge($middlewares, $registeredMiddlewares);
            }

            $psr7Response = $this->dispatcher->dispatch($psr7Request, $middlewares, $this->coreMiddleware);
        } catch (Throwable $throwable) {
            // Delegate the exception to exception handler.
            $psr7Response = $this->exceptionHandlerDispatcher->dispatch($throwable, $this->exceptionHandlers);
        } finally {
            // Send the Response to client.
            if (! isset($psr7Response)) {
                return;
            }
            if (isset($psr7Request) && $psr7Request->getMethod() === 'HEAD') {
                $this->responseEmitter->emit($psr7Response, $response, false);
            } else {
                $this->responseEmitter->emit($psr7Response, $response, true);
            }
        }
    }

    public function getServerName(): string
    {
        return $this->serverName;
    }

    /**
     * @return $this
     */
    public function setServerName(string $serverName): self
    {
        $this->serverName = $serverName;
        return $this;
    }

    protected function createDispatcher(string $serverName): Dispatcher
    {
        $factory = $this->container->get(DispatcherFactory::class);
        return $factory->getDispatcher($serverName);
    }

    protected function getDefaultExceptionHandler(): array
    {
        return [
            HttpExceptionHandler::class,
        ];
    }

    protected function createCoreMiddleware(): CoreMiddlewareInterface
    {
        return make(CoreMiddleware::class, [$this->container, $this->serverName]);
    }

    /**
     * Initialize PSR-7 Request and Response objects.
     * @param mixed $request swoole request or psr server request
     * @param mixed $response swoole response or swow session
     */
    protected function initRequestAndResponse(mixed $request, mixed $response): array
    {
        Context::set(ResponseInterface::class, $psr7Response = new Psr7Response());

        if ($request instanceof ServerRequestInterface) {
            $psr7Request = $request;
        } else {
            $psr7Request = Psr7Request::loadFromSwooleRequest($request);
        }
        Context::set(ServerRequestInterface::class, $psr7Request);
        return [$psr7Request, $psr7Response];
    }
}
