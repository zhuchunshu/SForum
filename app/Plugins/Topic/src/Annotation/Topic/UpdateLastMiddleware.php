<?php

namespace App\Plugins\Topic\src\Annotation\Topic;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;
/**
 * @Annotation
 * @Target({"CLASS"})
 */
#[Attribute(Attribute::TARGET_CLASS)]
class UpdateLastMiddleware extends AbstractAnnotation
{
}