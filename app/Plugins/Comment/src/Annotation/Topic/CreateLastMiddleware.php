<?php

namespace App\Plugins\Comment\src\Annotation\Topic;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;
/**
 * @Annotation
 * @Target({"CLASS"})
 */
#[Attribute(Attribute::TARGET_CLASS)]
class CreateLastMiddleware extends AbstractAnnotation
{
}