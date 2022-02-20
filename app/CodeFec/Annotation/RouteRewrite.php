<?php

namespace App\CodeFec\Annotation;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;

/**
 * @Annotation
 * @Target({"CLASS","METHOD"})
 */
#[Attribute(Attribute::TARGET_CLASS | Attribute::TARGET_METHOD)]
class RouteRewrite extends AbstractAnnotation
{
	public string $route;
	public string $callback='handler';
}