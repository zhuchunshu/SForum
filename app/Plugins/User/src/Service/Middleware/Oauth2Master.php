<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Service\Middleware;

use App\Model\AdminOption;
use App\Plugins\User\src\Service\interfaces\Oauth2SettingInterface;

class Oauth2Master implements Oauth2SettingInterface
{
    public function handler(array $data, \Closure $next)
    {
        if (! arr_has($data, 'enable')) {
            $data['enable'] = [];
        }
        $enable = json_encode($data['enable'], JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE);
        $name = ['name' => 'oauth2_enable'];
        $values = ['value' => $enable];
        AdminOption::query()->updateOrInsert($name, $values);
        options_clear();
        return $next($data);
    }
}
