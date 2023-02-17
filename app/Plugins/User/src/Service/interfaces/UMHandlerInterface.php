<?php

namespace App\Plugins\User\src\Service\interfaces;

interface UMHandlerInterface
{
    public function handler(array $data, \Closure $next);
}