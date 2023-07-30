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

use Hyperf\Di\Annotation\AnnotationCollector;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

class BeforeMiddleware implements MiddlewareInterface
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
        foreach (AnnotationCollector::getClassesByAnnotation(\App\CodeFec\Annotation\BeforeMiddleware::class) as $classes => $v) {
            $obj = new $classes($this->container);
            if (method_exists($obj, 'process')) {
                return $obj->process($request, $handler);
            }
        }
        return $handler->handle($request);
    }
}
