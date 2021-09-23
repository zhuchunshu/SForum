<?php

use App\Plugins\Core\src\Lib\Authority\Authority;
use App\Plugins\Core\src\Lib\ShortCode\ShortCode;
use App\Plugins\Core\src\Lib\ShortCodeR\ShortCodeR;
use DivineOmega\PHPSummary\SummaryTool;
use JetBrains\PhpStorm\Pure;
use App\Plugins\Core\src\Lib\Redirect;
use App\Plugins\Core\src\Lib\UserVerEmail;

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

if(!function_exists("super_avatar")){
    function super_avatar($user_data): string
    {
        if($user_data->avatar){
            return $user_data->avatar;
        }

        if(get_options("core_user_def_avatar","gavatar")!=="multiavatar") {
            return get_options("theme_common_gavatar", "https://cn.gravatar.com/avatar/") . md5($user_data->email);
        }
        return "/user/multiavatar/".$user_data->username."/avatar.jpg";
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

if(!function_exists("avatar_url")){
    function avatar_url(int $user_id): string
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
            return $ud->avatar;

        }else{
            if(get_options("core_user_def_avatar","gavatar")!=="multiavatar"){
                $url = get_options("theme_common_gavatar","https://cn.gravatar.com/avatar/").md5($ud->email);
                return $url;
            }else{
                return "/user/multiavatar/".$ud->id;
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

if(!function_exists("core_Str_menu_url")){
    function core_Str_menu_url(string $path): string
    {
        if($path ==="//"){
            $path = "/";
        }
        return $path;
    }
}

if (!function_exists("core_menu_pd")) {
    function core_menu_pd(string $id)
    {
        foreach (Itf()->get('menu') as $value) {
            if (arr_has($value, "parent_id") && "menu_" . $value['parent_id'] === (string)$id) {
                return true;
            }
        }
        return false;
    }
}

if(!function_exists("core_Itf_id")){
    function core_Itf_id($name,$id){
        return \Hyperf\Utils\Str::after($id,$name."_");
    }
}

if (!function_exists("core_menu_pdArr")) {
    function core_menu_pdArr($id): array
    {
        $arr = [];
        foreach (Itf()->get("menu") as $key => $value) {
            if (arr_has($value, "parent_id") && "menu_".$value['parent_id'] === $id) {
                $arr[$key] = $value;
            }
        }
        return $arr;
    }
}

if(!function_exists("core_default")){
    function core_default($string=null,$default=null){
        if($string){
            return $string;
        }
        return $default;
    }
}

if(!function_exists("markdown")){
    function markdown(): \Parsedown
    {
        return new Parsedown();
    }
}

if(!function_exists("ShortCode")){
    function ShortCode(): ShortCode
    {
        return new ShortCode();
    }
}

if(!function_exists("ShortCodeR")){
    function ShortCodeR(): ShortCodeR
    {
        return new ShortCodeR();
    }
}

if(!function_exists("xss")){
    function xss(): \App\Plugins\Core\src\Lib\Xss\Xss
    {
        return new App\Plugins\Core\src\Lib\Xss\Xss();
    }
}

if(!function_exists("summary")){
    function summary($content): string
    {
        return (new SummaryTool($content))->getSummary();
    }
}

if(!function_exists("deOptions")){
    function deOptions($json){
        return json_decode($json, true);
    }
}

if(!function_exists("getAllImg")){
    function getAllImg($content):array{
        $preg = '/<img.*?src=[\"|\']?(.*?)[\"|\']?\s.*?>/i';//匹配img标签的正则表达式

        preg_match_all($preg, $content, $allImg);//这里匹配所有的imgecho
        return $allImg[1];
    }
}

if(!function_exists("format_date")){
    function format_date($time)
    {
        $t = time() - strtotime($time);
        $f = array(
            '31536000' => '年',
            '2592000' => '个月',
            '604800' => '星期',
            '86400' => '天',
            '3600' => '小时',
            '60' => '分钟',
            '1' => '秒'
        );
        foreach ($f as $k => $v) {
            if (0 != $c = floor($t / (int)$k)) {
                return $c . $v . '前';
            }
        }
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

if(!function_exists("remove_bbCode")){
    function remove_bbCode($content){
        $pattern = "/\[(.*?)\](.*?)\[\/(.*?)\]/is";

        $content = preg_replace_callback($pattern, function($match){
            return "";
        },$content);
        $content = str_replace(" ","",$content);
        return $content;
    }
}

if(!function_exists("Authority")){
    function Authority(){
        return new Authority();
    }
}

if(!function_exists("curd")){
    function curd(): \App\Plugins\Core\src\Lib\Curd
    {
        return new \App\Plugins\Core\src\Lib\Curd();
    }
}

if(!function_exists("core_http_build_query")){
    function core_http_build_query(array $data,array $merge): string
    {
        $data = array_merge($data,$merge);
        return http_build_query($data);
    }
}

if(!function_exists("core_http_url")){
    function core_http_url(): string
    {
        $query = http_build_query(request()->all());
        return request()->path()."?".$query;
    }
}

if(!function_exists("core_get_page")){
    function core_get_page(string $url): array
    {
        $data = explode("=",parse_url($url)['query']);
        return [$data[0] => $data[1]];
    }
}