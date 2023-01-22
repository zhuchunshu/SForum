<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */

namespace App\Plugins\User\src\Service\interfaces;

use App\CodeFec\storage\Oauth2Handler;

class AbstractOauth2 implements Oauth2Interface
{
    public function admin_view():string
    {
        return 'User::Admin.oauth2.demo';
    }

    public function mark(): string
    {
        return 'mark';
    }


    public function name(): string
    {
        return '在前台显示的名称';
    }

    public function view(): string
    {
        return 'User::Admin.oauth2.demo';
    }

    public function icon(): string
    {
        return '';
    }

    public function setting_handler(): string
    {
        return Oauth2Handler::class;
    }
}
