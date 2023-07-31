<?php

namespace App\Plugins\User\src\Annotation;

use Attribute;
use Hyperf\Di\Annotation\AbstractAnnotation;
/**
 * @Annotation
 * @Target({"CLASS"})
 */
#[Attribute(Attribute::TARGET_CLASS)]
class UserManagementAnnotation extends AbstractAnnotation
{
}