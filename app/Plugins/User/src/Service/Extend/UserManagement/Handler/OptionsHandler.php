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
use App\Plugins\User\src\Models\UsersOption;
use App\Plugins\User\src\Service\interfaces\UMHandlerInterface;

class OptionsHandler implements UMHandlerInterface
{
    public function handler(array $data, \Closure $next)
    {
        $id = $data['id'];
        if (arr_has($data, 'options')) {
            $options_id = User::query()->where('id', $id)->value('options_id');
            $options = UsersOption::find($options_id);
            foreach ($data['options'] as $k => $v) {
                $options->{$k} = trim($v);
                $options->save();
            }
        }
        return $next($data);
    }
}
