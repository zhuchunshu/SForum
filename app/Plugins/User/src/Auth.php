<?php

namespace App\Plugins\User\src;

use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
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
        session()->remove('auth_data_class');
        session()->remove('auth_data');
        return true;
    }

    public function data()
    {
        if(!session()->has("auth_data")){
            session()->set("auth_data",User::query()->where("id",session()->get('auth'))->with("Class")->first());
        }
        return session()->get("auth_data");
    }

    public function Class(){
        if(!session()->has("auth_data_class")){
            session()->set("auth_data_class",UserClass::query()->where("id",auth()->data()->class_id)->first());
        }
        return session()->get("auth_data_class");
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