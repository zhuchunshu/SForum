<?php
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