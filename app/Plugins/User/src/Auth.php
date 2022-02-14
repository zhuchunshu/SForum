<?php

namespace App\Plugins\User\src;

use App\Plugins\User\src\Event\AfterLogin;
use App\Plugins\User\src\Event\Logout;
use App\Plugins\User\src\Lib\UserAuth;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\Utils\Str;
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
            $token = Str::random(17);
            session()->set('auth', $token);
            (new UserAuth())->create($user->id,$token);
            session()->set("auth_data",User::query()->where("id",$this->id())->with("Class")->first());
            session()->set("auth_data_class",UserClass::query()->where("id",auth()->data()->class_id)->first());
            session()->set("auth_data_options",UsersOption::query()->where("id",auth()->data()->options_id)->first());
	        EventDispatcher()->dispatch(new AfterLogin($user));
            return true;
        }
        return false;
    }

    public function token(){
        if($this->check()===true){
            return session()->get("auth",null);
        }
        return null;
    }

    public function logout(): bool
    {
		EventDispatcher()->dispatch(new Logout($this->id()));
        (new UserAuth())->destroy_token(session()->get('auth'));
        session()->remove('auth');
        session()->remove('auth_data_class');
        session()->remove('auth_data_options');
        session()->remove('auth_data');
        return true;
    }

    public function data()
    {
        if(!session()->has("auth_data")){
            session()->set("auth_data",User::query()->where("id",$this->id())->with("Class")->first());
        }
        return session()->get("auth_data");
    }

    public function Class(){
        if(!session()->has("auth_data_class")){
            session()->set("auth_data_class",UserClass::query()->where("id",auth()->data()->class_id)->first());
        }
        return session()->get("auth_data_class");
    }

    public function Options(){
        if(!session()->has("auth_data_options")){
            session()->set("auth_data_options",UsersOption::query()->where("id",auth()->data()->options_id)->first());
        }
        return session()->get("auth_data_options");
    }

    public function UpdateClass(){
        session()->set("auth_data_class",UserClass::query()->where("id",auth()->data()->class_id)->first());
    }

    public function UpdateOptions(){
        session()->set("auth_data_options",UsersOption::query()->where("id",auth()->data()->options_id)->first());
    }

    public function id()
    {
        return (int)@UsersAuth::query()->where("token",session()->get('auth'))->first('user_id')->user_id;
    }

    public function check(): bool
    {
        if(!session()->has('auth')){
            return false;
        }
        if(User::query()->where("id",$this->id())->count()){
            return true;
        }

        return false;
    }
}