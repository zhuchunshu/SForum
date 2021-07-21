<?php

namespace App\Plugins\User\src;

use App\Plugins\User\src\Models\User;
use HyperfExt\Hashing\Hash;

class Auth
{
    public static function SignIn(string $email, string $password): bool
    {
        if (! User::query()->where('email', $email)->count()) {
            return false;
        }
        // 数据库里的密码
        $user = User::query()->where('email', $email)->first();
        if (Hash::check($password, $user->password)) {
            session()->set('auth', $user->id);
            return true;
        }
        return false;
    }

    public static function data()
    {
        return User::query()->where("id",session()->get('auth'))->first();
    }

    public static function id()
    {
        return session()->get('auth');
    }

    public static function check(): bool
    {
        if(!session()->has('auth')){
            return false;
        }
        if(User::query()->where("id",session()->get('auth'))->count()){
            return true;
        }else{
            return false;
        }
    }
}