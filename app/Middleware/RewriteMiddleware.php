<?php

declare(strict_types=1);

namespace App\Middleware;

use App\CodeFec\Annotation\RouteRewrite;
use App\Controller\AdminController;
use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\HttpServer\Router\Dispatched;
use Hyperf\HttpServer\Router\Handler;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

class RewriteMiddleware implements MiddlewareInterface
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
        /** @var Dispatched $dispatched */
        $dispatched = $request->getAttribute(Dispatched::class);

        // 安全重写,没有不新建
        foreach (Router()->get() as $key => $value) {
            if (@$dispatched->handler->route === $key) {
                $dispatched->handler->callback = $value;
            }
        }
	
	    $Route_Class = AnnotationCollector::getClassesByAnnotation(RouteRewrite::class);
		foreach ($Route_Class as $key=>$data){
			$callback = [$key,$data->callback];
			if (@$dispatched->handler->route === $data->route) {
				$dispatched->handler->callback = $callback;
			}
		}
	
	    $routeMethods = AnnotationCollector::getMethodsByAnnotation(RouteRewrite::class);
	    foreach($routeMethods as $data){
		    $callback = [$data['class'],$data['method']];
		    if (@$dispatched->handler->route === $data['annotation']->route) {
			    $dispatched->handler->callback = $callback;
		    }
	    }
        return $handler->handle($request);

    }
}