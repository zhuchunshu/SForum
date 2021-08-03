<?php

use App\Plugins\Core\src\Lib\Redirect;
use App\Plugins\Core\src\Lib\UserVerEmail;
use JetBrains\PhpStorm\Pure;

if(!function_exists("plugins_core_common_theme")){
    function plugins_core_common_theme($default="light"){
        if(!\App\Model\AdminOption::query()->where("name","theme_common_theme")->count()){
            return $default;
        }
        return \App\Model\AdminOption::query()->where("name","theme_common_theme")->first()->value;
    }
}

if(!function_exists("plugins_core_theme")){
    function plugins_core_theme(){
        if(session()->has("core.common.theme")){
            return session()->get("core.common.theme");
        }
        return plugins_core_common_theme();
    }
}

if(!function_exists("plugins_core_captcha")){
    function plugins_core_captcha(): \App\Plugins\Core\src\Lib\Captcha
    {
        return new \App\Plugins\Core\src\Lib\Captcha();
    }
}

if(!function_exists("plugins_core_captcha")){
    function plugins_core_captcha(): \App\Plugins\Core\src\Lib\Captcha
    {
        return new \App\Plugins\Core\src\Lib\Captcha();
    }
}

if(!function_exists("plugins_core_user_reg_defuc")){
    function plugins_core_user_reg_defuc()
    {
        return \App\Plugins\User\src\Models\UserClass::query()->select("id","name")->get();
    }
}

if(!function_exists("avatar")){
    function avatar(int $user_id,$class=null): string
    {
        $time = get_options("core_user_def_avatar_cache",600);
        if(get_options("core_user_avatar_cache","1")==="1"){
            if(cache()->has("core.avatar.".$user_id)){
                $ud = cache()->get("core.avatar.".$user_id);
            }else{
                $ud = \App\Plugins\User\src\Models\User::query()->where("id",$user_id)->first();
                cache()->set("core.avatar.".$user_id,$ud,$time);
            }
        }else{
            $ud = \App\Plugins\User\src\Models\User::query()->where("id",$user_id)->first();
        }

        if($ud->avatar){
            return <<<HTML
<span class="avatar {$class}" style="background-image: url({$ud->avatar})"></span>
HTML;

        }else{
            if(get_options("core_user_def_avatar","gavatar")!=="multiavatar"){
                $url = get_options("theme_common_gavatar","https://cn.gravatar.com/avatar/").md5($ud->email);
            return <<<HTML
<span class="avatar {$class}" style="background-image: url({$url})"></span>
HTML;
            }else{
                $img = new Multiavatar();
                $img = $img($ud->username, null, null);
                return <<<HTML
<span class="avatar {$class}">{$img}</span>
HTML;
            }
        }
    }
}


if(!function_exists("redirect")){
    #[Pure] function redirect(): Redirect
    {
        return new Redirect();
    }
}

if(!function_exists("core_user_ver_email_make")){
    function core_user_ver_email(): UserVerEmail
    {
        return new UserVerEmail();
    }
}

if(!function_exists("Core_Ui")){
    function Core_Ui(): \App\Plugins\Core\src\Lib\Ui
    {
        return new App\Plugins\Core\src\Lib\Ui();
    }
}