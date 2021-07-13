<?php

namespace App\Plugins\User\src;

use App\Plugins\User\src\Models\User;
use HyperfExt\Hashing\Hash;

class Auth
{
    public static function SignIn(string $username, string $password): bool
    {
        if (! User::query()->where('username', $username)->count()) {
            return false;
        }
        // 数据库里的密码
        $user = User::query()->where('username', $username)->first();
        if (Hash::check($password, $user->password)) {
            session()->set('admin', $user->id);
            return true;
        }
        return false;
    }

    public static function data()
    {
        return User::query()->where("id",session()->get('admin'))->first();
    }

    public static function id()
    {
        return session()->get('admin');
    }

    public static function check(): bool
    {
        if(!session()->has('admin')){
            return false;
        }
        if(User::query()->where("id",session()->get('admin'))->count()){
            return true;
        }else{
            return false;
        }
    }
}