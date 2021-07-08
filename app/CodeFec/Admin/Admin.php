<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace App\CodeFec\Admin;

use App\Model\AdminUser;
use HyperfExt\Hashing\Hash;

class Admin
{
    public static function SignIn(string $username, string $password): bool
    {
        if (! AdminUser::query()->where('username', $username)->count()) {
            return false;
        }
        // 数据库里的密码
        $user = AdminUser::query()->where('username', $username)->first();
        if (Hash::check($password, $user->password)) {
            session()->set('admin', $user);
            return true;
        }
        return false;
    }

    public static function data()
    {
        return session()->get('admin');
    }

    public static function id()
    {
        return session()->get('admin')['id'];
    }

    public static function Check(): bool
    {
        return session()->has('admin');
    }
}
