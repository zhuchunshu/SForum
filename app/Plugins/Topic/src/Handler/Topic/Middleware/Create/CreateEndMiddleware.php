<?php

namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

class CreateEndMiddleware implements MiddlewareInterface
{
    public function handler($data,\Closure $next)
    {
        return redirect()->url('/')->with('success', '发表成功!')->go();
    }
}