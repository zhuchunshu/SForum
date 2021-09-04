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

if(!function_exists("replace_all_at_space")){
    function replace_all_at_space(string $content): string
    {
        //$pattern = "/\\$\\[(.*?)]/u";
        $pattern = "/@(.*?)[^ <\/p>]+/u";
        return preg_replace_callback($pattern, static function($match){
            return $match[0]." ";
        },$content);
    }
}

if(!function_exists("remove_all_p_space")){
    function remove_all_p_space(string $content):string{
        return str_replace(" </p>","</p>",$content);
    }
}

if(!function_exists("replace_all_at")){
    function replace_all_at(string $content):string{
        //$pattern = "/\\$\\[(.*?)]/u";
        $pattern = "/@(.*?)[^ ]+/u";
        $content = replace_all_at_space($content);
        return remove_all_p_space(preg_replace_callback($pattern, static function($match){
            return (new \App\Plugins\Core\src\Lib\TextParsing())->at($match[0]);
        },$content));
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

    function replace_all_keywords(string $content):string{
        $pattern = "/\\$\\[(.*?)]/u";
        return preg_replace_callback($pattern, static function($match){
            return (new \App\Plugins\Core\src\Lib\TextParsing())->keywords($match[1]);
        },$content);
    }
}