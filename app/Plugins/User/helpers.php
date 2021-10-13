<?php
if(!function_exists("auth")){
    function auth(): \App\Plugins\User\src\Auth
    {
        return new \App\Plugins\User\src\Auth();
    }
}

if(!function_exists("user_avatar")){
    function user_avatar($email,$avatar=null){
        if($avatar){
            return $avatar;
        }
        return get_options("theme_common_gavatar","https://cn.gravatar.com/avatar/").md5($email);
    }
}

if(!function_exists("user_notice")){
    function user_notice(): \App\Plugins\User\src\Lib\UserNotice
    {
        return new \App\Plugins\User\src\Lib\UserNotice();
    }
}
