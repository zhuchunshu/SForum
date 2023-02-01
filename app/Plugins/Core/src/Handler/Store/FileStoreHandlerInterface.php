<?php

namespace App\Plugins\Core\src\Handler\Store;

interface FileStoreHandlerInterface
{
    public function handler(array $data, \Closure $next);
}