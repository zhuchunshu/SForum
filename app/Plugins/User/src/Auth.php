<?php

namespace App\Plugins\User\src;

use App\Plugins\User\src\Models\User;
use HyperfExt\Hashing\Hash;

class Auth
{
    public function SignIn(string $email, string $password): bool
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

    public function logout(): bool
    {
        session()->remove('auth');
        session()->remove('auth_data');
        return true;
    }

    public function data()
    {
        if(!session()->has("auth_data")){
            session()->set("auth_data",User::query()->where("id",session()->get('auth'))->first());
        }
        return session()->get("auth_data");
    }

    public function id()
    {
        return session()->get('auth');
    }

    public function check(): bool
    {
        if(!session()->has('auth')){
            return false;
        }
        if(User::query()->where("id",session()->get('auth'))->count()){
            return true;
        }

        return false;
    }
}