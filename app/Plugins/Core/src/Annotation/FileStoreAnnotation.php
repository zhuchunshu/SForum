<?php

namespace App\Plugins\Core\src\Annotation;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;
/**
 * @Annotation
 * @Target({"CLASS"})
 */
#[Attribute(Attribute::TARGET_CLASS)]
class FileStoreAnnotation extends AbstractAnnotation
{
}