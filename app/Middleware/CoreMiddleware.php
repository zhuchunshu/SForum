<?php
declare(strict_types=1);

namespace App\Middleware;

use Hyperf\HttpMessage\Stream\SwooleStream;
use Closure;
use FastRoute\Dispatcher;
use Hyperf\Contract\NormalizerInterface;
use Hyperf\Di\ClosureDefinitionCollectorInterface;
use Hyperf\Di\MethodDefinitionCollectorInterface;
use Hyperf\HttpMessage\Exception\MethodNotAllowedHttpException;
use Hyperf\HttpMessage\Exception\NotFoundHttpException;
use Hyperf\HttpMessage\Exception\ServerErrorHttpException;
use Hyperf\HttpServer\Contract\CoreMiddlewareInterface;
use Hyperf\HttpServer\Router\Dispatched;
use Hyperf\HttpServer\Router\DispatcherFactory;
use Hyperf\RateLimit\Annotation\RateLimit;
use Hyperf\Server\Exception\ServerException;
use Hyperf\Utils\Codec\Json;
use Hyperf\Utils\Context;
use Hyperf\Utils\Contracts\Jsonable;
use Psr\Container\ContainerInterface;
use Psr\Http\Server\RequestHandlerInterface;
use Hyperf\Utils\Contracts\Arrayable;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

class CoreMiddleware extends \Hyperf\HttpServer\CoreMiddleware
{
    /**
     * Handle the response when cannot found any routes.
     *
     * @return array|Arrayable|mixed|ResponseInterface|string
     */
    protected function handleNotFound(ServerRequestInterface $request)
    {
        // 重写路由找不到的处理逻辑
        return admin_abort(["msg" => "页面不存在"],404);
    }

    /**
     * Handle the response when the routes found but doesn't match any available methods.
     *
     * @param array $methods
     * @param ServerRequestInterface $request
     * @return ResponseInterface
     */
    protected function handleMethodNotAllowed(array $methods, ServerRequestInterface $request): ResponseInterface
    {
        // 重写 HTTP 方法不允许的处理逻辑
        return $this->response()->withStatus(405);
    }
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
        return $response->withAddedHeader('Server', config("codefec.app.name"));
    }
}
