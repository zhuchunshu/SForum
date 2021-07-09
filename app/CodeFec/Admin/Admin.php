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
            session()->set('admin', $user->id);
            return true;
        }
        return false;
    }

    public static function data()
    {
        return AdminUser::query()->where("id",session()->get('admin'))->first();
    }

    public static function id()
    {
        return session()->get('admin');
    }

    public static function Check(): bool
    {
        if(!session()->has('admin')){
            return false;
        }
        if(AdminUser::query()->where("id",session()->get('admin'))->count()){
            return true;
        }else{
            return false;
        }
    }
}
