<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Service\Extend\UserManagement\Handler;

use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Service\interfaces\UMHandlerInterface;

class BasisHandler implements UMHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        $id = $data['id'];

        if (arr_has($data, 'basis')) {
            $user = User::query()->find($id);
            foreach ($data['basis'] as $k => $v) {
                $user->{$k} = trim($v);
                $user->save();
            }
        }

        return $next($data);
    }
}
