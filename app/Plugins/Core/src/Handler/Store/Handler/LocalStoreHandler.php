<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Handler\Store\Handler;

use App\Plugins\Core\src\Handler\Store\FileStoreHandlerInterface;

class LocalStoreHandler implements FileStoreHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        return $next($data);
    }
}
