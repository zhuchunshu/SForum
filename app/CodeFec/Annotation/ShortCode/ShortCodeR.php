<?php

namespace App\CodeFec\Annotation\ShortCode;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;

/**
 * @Annotation
 * @Target({"METHOD"})
 */
#[Attribute(Attribute::TARGET_METHOD)]
class ShortCodeR extends AbstractAnnotation
{
	public string $name;
}
