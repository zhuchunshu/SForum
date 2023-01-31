<?php

namespace App\Plugins\Mail\src;

interface SendServiceHandlerInterface
{
    public function handler(array $data, \Closure $next);
}