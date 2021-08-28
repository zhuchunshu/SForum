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


if(!function_exists("get_all_at")){
    /**
     * 获取内容中所有被艾特的用户
     * @param string $content
     * @return array
     */
    function get_all_at(string $content):array{
        preg_match_all("/(?<=@)[^ ]+/u", $content, $arr);
        return $arr[0];
    }
}

if(!function_exists("get_all_keywords")){

    /**
     * 获取内容中所有话题关键词
     * @param string $content
     * @return array
     */
    function get_all_keywords(string $content):array{
        preg_match_all("/(?<=\\$\\[)[^]]+/u", $content, $arrMatches);
        return $arrMatches[0];
    }
}