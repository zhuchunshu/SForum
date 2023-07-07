<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User;

/**
 * Class User.
 * @see https://github.com/zhuchunshu/sf-user
 * @name User
 * @version 1.0.0
 */
class User
{
    public function handler(): void
    {
        $this->boot();
        $this->helpers();
        $this->menu();
        $this->hook();
    }

    public function menu()
    {
        include __DIR__ . '/menu.php';
    }

    public function helpers()
    {
        include __DIR__ . '/helpers.php';
    }

    public function boot()
    {
        include __DIR__ . '/bootstrap.php';
    }

    private function hook()
    {
        include __DIR__ . '/hook.php';
    }
}
