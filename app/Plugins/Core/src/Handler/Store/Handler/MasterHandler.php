<?php

namespace App\Plugins\Core\src\Handler\Store\Handler;

use App\Plugins\Core\src\Handler\Store\FileStoreHandlerInterface;
use Hyperf\Utils\Arr;

class MasterHandler implements FileStoreHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        if (Arr::has($data, 'service') && @$data['service']) {
            set_options('file_store_service', $data['service']);
        }
        return $next($data);
    }
}