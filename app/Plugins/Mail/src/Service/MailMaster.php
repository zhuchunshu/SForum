<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\Service;

use App\Plugins\Mail\src\SendServiceHandlerInterface;
use Hyperf\Collection\Arr;
class MailMaster implements SendServiceHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        if (Arr::has($data, 'service') && @$data['service']) {
            set_options('mail_service', $data['service']);
        }
        return $next($data);
    }
}