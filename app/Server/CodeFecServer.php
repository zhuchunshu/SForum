<?php
namespace App\Server;

use App\CodeFec\CodeFec;
use Throwable;
use FastRoute\Dispatcher;
use Hyperf\Utils\Context;
use Hyperf\Contract\ConfigInterface;
use Hyperf\Dispatcher\HttpDispatcher;
use Hyperf\HttpServer\CoreMiddleware;
use Psr\Container\ContainerInterface;
use Hyperf\HttpServer\ResponseEmitter;
use Hyperf\Contract\OnRequestInterface;
use Hyperf\Utils\Coordinator\Constants;
use Psr\Http\Message\ResponseInterface;
use Hyperf\HttpServer\MiddlewareManager;
use Hyperf\HttpServer\Router\Dispatched;
use Psr\Http\Message\ServerRequestInterface;
use Hyperf\HttpServer\Router\DispatcherFactory;
use Hyperf\Utils\Coordinator\CoordinatorManager;
use Hyperf\Contract\MiddlewareInitializerInterface;
use Hyperf\HttpMessage\Server\Request as Psr7Request;
use Hyperf\ExceptionHandler\ExceptionHandlerDispatcher;
use Hyperf\HttpMessage\Server\Response as Psr7Response;
use Hyperf\HttpServer\Contract\CoreMiddlewareInterface;
use Hyperf\HttpServer\Exception\Handler\HttpExceptionHandler;

class CodeFecServer implements OnRequestInterface, MiddlewareInitializerInterface {

    /**
     * @var ContainerInterface
     */
    protected ContainerInterface $container;

    /**
     * @var HttpDispatcher
     */
    protected HttpDispatcher $dispatcher;

    /**
     * @var ExceptionHandlerDispatcher
     */
    protected ExceptionHandlerDispatcher $exceptionHandlerDispatcher;

    /**
     * @var array
     */
    protected array $middlewares;

    /**
     * @var CoreMiddlewareInterface
     */
    protected CoreMiddlewareInterface $coreMiddleware;

    /**
     * @var array
     */
    protected array $exceptionHandlers;

    /**
     * @var Dispatcher
     */
    protected Dispatcher $routerDispatcher;

    /**
     * @var ResponseEmitter
     */
    protected ResponseEmitter $responseEmitter;

    /**
     * @var string
     */
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
        (new CodeFec)->handle();
        Context::set(ServerRequestInterface::class, $psr7Request);
        return [$psr7Request, $psr7Response];
    }

}