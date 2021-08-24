<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */

use App\Model\AdminPlugin;
use Hyperf\Utils\Context;
use App\Model\AdminOption;
use Illuminate\Support\Arr;
use Illuminate\Support\Str;
use App\CodeFec\Admin\Admin;
use App\CodeFec\Itf\Setting\SettingInterface;
use Hyperf\Paginator\UrlWindow;
use Hyperf\View\RenderInterface;
use App\CodeFec\Menu\MenuInterface;
use Hyperf\Utils\ApplicationContext;
use Illuminate\Support\Facades\File;
use Hyperf\Contract\SessionInterface;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ServerRequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;

function public_path($path = ''): string
{
    if ($path != '') {
        return config('server.settings.document_root') . '/' . $path;
    }
    return config('server.settings.document_root');
}

if (!function_exists('mix_manifest')) {
    function mix_manifest()
    {
        return file_get_contents(public_path('mix-manifest.json'));
    }
}

if (!function_exists('mix')) {
    function mix($path)
    {
        $list = mix_manifest();
        $result = json_decode($list, true);
        if (Arr::has($result, '/' . $path)) {
            return $result['/' . $path];
        }
        return "/".$path;
    }
}

if (!function_exists("arr_has")) {
    function arr_has($array, $keys)
    {
        return Arr::has($array, $keys);
    }
}

/**
 * 容器实例
 */
if (!function_exists('container')) {
    function container()
    {
        return ApplicationContext::getContainer();
    }
}

/**
 * redis 客户端实例
 */
if (!function_exists('redis')) {
    function redis()
    {
        return container()->get(Redis::class);
    }
}

/**
 * server 实例 基于 swoole server
 */
if (!function_exists('server')) {
    function server()
    {
        return container()->get(ServerFactory::class)->getServer()->getServer();
    }
}

/**
 * websocket frame 实例
 */
if (!function_exists('frame')) {
    function frame()
    {
        return container()->get(Frame::class);
    }
}

/**
 * websocket 实例
 */
if (!function_exists('websocket')) {
    function websocket()
    {
        return container()->get(WebSocketServer::class);
    }
}

/**
 * 缓存实例 简单的缓存
 */
if (!function_exists('cache')) {
    function cache()
    {
        return container()->get(Psr\SimpleCache\CacheInterface::class);
    }
}

/**
 * 控制台日志
 */
if (!function_exists('stdLog')) {
    function stdLog()
    {
        return container()->get(StdoutLoggerInterface::class);
    }
}

/**
 * 文件日志
 */
if (!function_exists('logger')) {
    function logger()
    {
        return container()->get(LoggerFactory::class)->make();
    }
}

if (!function_exists('response')) {
    function response()
    {
        return container()->get(ResponseInterface::class);
    }
}

if (!function_exists("request")) {
    function request()
    {
        return new Hyperf\HttpServer\Request();
    }
}

if (!function_exists("path_class")) {
    function path_class()
    {
        $path = request()->path();
        $result = str_replace("/", "-", $path);
        $result = Str::before($result, '.');
        if ($result == "-") {
            return "main";
        }
        return $result;
    }
}


if (!function_exists("menu")) {
    function menu()
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(MenuInterface::class);
    }
}

if (!function_exists("view")) {
    function view(string $view, array $data = [], int $code = 200): \Psr\Http\Message\ResponseInterface
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(RenderInterface::class)->render($view, $data, $code);
    }
}

if (!function_exists("menu_pd")) {
    function menu_pd($id)
    {
        $i = 0;
        foreach (menu()->get() as $key => $value) {
            if (arr_has($value, "parent_id")) {
                if ($value['parent_id'] === $id) {
                    $i++;
                }
            }
        }
        return $i;
    }
}

if (!function_exists("menu_pdArr")) {
    function menu_pdArr($id)
    {
        $arr = [];
        foreach (menu()->get() as $key => $value) {
            if (arr_has($value, "parent_id")) {
                if ($value['parent_id'] == $id) {
                    $arr[$key] = $value;
                }
            }
        }
        return $arr;
    }
}

if (!function_exists("Json_Api")) {
    function Json_Api(int $code = 200, bool $success = true, array $result = [])
    {
        return [
            "code" => $code,
            "success" => $success,
            "result" => $result
        ];
    }
}

if (!function_exists("session")) {
    function session()
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(SessionInterface::class);
    }
}

// 获取目录下的所有文件夹
if (!function_exists("getPath")) {
    function getPath($path)
    {
        if (!is_dir($path)) {
            return false;
        }
        $arr = array();
        $data = scandir($path);
        foreach ($data as $value) {
            if ($value != '.' && $value != '..') {
                $arr[] = $value;
            }
        }
        return $arr;
    }
}

if (!function_exists("plugin_path")) {
    function plugin_path($path = null)
    {
        if (!$path) {
            return BASE_PATH . "/app/Plugins";
        }
        return BASE_PATH . "/app/Plugins/" . $path;
    }
}

if (!function_exists("read_file")) {
    function read_file($file_path)
    {
        if (file_exists($file_path)) {
            $str = File::get($file_path);
            return $str;
        } else {
            return null;
        }
    }
}

if (!function_exists("read_plugin_data")) {
    /**
     * 读取插件data.json文件
     *
     * @param string 插件目录名 $name
     */
    function read_plugin_data(string $name, $bool = true)
    {
        if ($bool === true) {
            return json_decode(@read_file(plugin_path($name . "/data.json")));
        } else {
            return json_decode(@read_file(plugin_path($name . "/data.json")), true);
        }
    }
}

if (!function_exists("admin_abort")) {
    /**
     * @param array|string $data
     * @param int $code
     * @return \Psr\Http\Message\ResponseInterface
     */
    function admin_abort(array|string $data, int $code = 403): \Psr\Http\Message\ResponseInterface
    {
        if(is_string($data)){
            $array = ['msg' => $data];
        }else{
            $array = $data;
        }
        if (request()->isMethod("POST") || request()->input("data") === "json") {
            return response()->json(Json_Api($code, false, $array));
        }
        return view('admin.error', [], $code);
    }
}

if (!function_exists("get_plugins_doc")) {
    function get_plugins_doc($class)
    {
        $re  = new ReflectionClass(new $class());
        $content = $re->getDocComment();
        $preg = "/@+(.*)/";
        preg_match_all($preg, $content, $result);
        $result = $result[1];
        $arr = [];
        foreach ($result as $key => $value) {
            $result1 = explode(" ", $value);
            $arr[$result1[0]] = $result1[1];
        }
        return $arr;
    }
}

if (!function_exists("deldir")) {
    function deldir($path)
    {
        //如果是目录则继续
        if (is_dir($path)) {
            //扫描一个文件夹内的所有文件夹和文件并返回数组
            $p = scandir($path);
            //如果 $p 中有两个以上的元素则说明当前 $path 不为空
            if (count($p) > 2) {
                foreach ($p as $val) {
                    //排除目录中的.和..
                    if ($val != "." && $val != "..") {
                        //如果是目录则递归子目录，继续操作
                        if (is_dir($path . $val)) {
                            //子目录中操作删除文件夹和文件
                            deldir($path . $val . '/');
                        } else {
                            //如果是文件直接删除
                            unlink($path . $val);
                        }
                    }
                }
            }
        }
        //删除目录
        return rmdir($path);
    }
}


if (!function_exists("copy_dir")) {
    function copy_dir($src, $dst)
    {
        $dir = opendir($src);
        @mkdir($dst);
        while (false !== ($file = readdir($dir))) {
            if (($file != '.') && ($file != '..')) {
                if (is_dir($src . '/' . $file)) {
                    copy_dir($src . '/' . $file, $dst . '/' . $file);
                    continue;
                } else {
                    copy($src . '/' . $file, $dst . '/' . $file);
                }
            }
        }
        closedir($dir);
    }
}

if (!function_exists('verify_ip')) {
    function verify_ip($realip)
    {
        return filter_var($realip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4);
    }
}

if (!function_exists('get_client_ip')) {
    function get_client_ip()
    {
        /**
         * @var ServerRequestInterface $request
         */
        $request = Context::get(ServerRequestInterface::class);
        $ip_addr = $request->getHeaderLine('x-forwarded-for');
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        $ip_addr = $request->getHeaderLine('remote-host');
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        $ip_addr = $request->getHeaderLine('x-real-ip');
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        $ip_addr = $request->getServerParams()['remote_addr'] ?? '0.0.0.0';
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        return '0.0.0.0';
    }
}

if(!function_exists("make_page")){
    function make_page($page,$default = "default"){
        $window = UrlWindow::make($page);

        $elements = array_filter([
            $window['first'],
            is_array($window['slider']) ? '...' : null,
            $window['slider'],
            is_array($window['last']) ? '...' : null,
            $window['last'],
        ]);
        return view("shared.page.".$default,['paginator' => $page,'elements' => $elements]);
    }
}

if (!function_exists("exhtml")) {
    function exhtml($descclear)
    {
        $descclear = str_replace("\r", "", $descclear); //过滤换行
        $descclear = str_replace("\n", "", $descclear); //过滤换行
        $descclear = str_replace("\t", "", $descclear); //过滤换行
        $descclear = str_replace("\r\n", "", $descclear); //过滤换行
        $descclear = preg_replace("/\s+/", " ", $descclear); //过滤多余回车
        $descclear = preg_replace("/<[ ]+/si", "<", $descclear); //过滤<__("<"号后面带空格)
        $descclear = preg_replace("/<\!--.*?-->/si", "", $descclear); //过滤html注释
        $descclear = preg_replace("/<(\!.*?)>/si", "", $descclear); //过滤DOCTYPE
        $descclear = preg_replace("/<(\/?html.*?)>/si", "", $descclear); //过滤html标签
        $descclear = preg_replace("/<(\/?head.*?)>/si", "", $descclear); //过滤head标签
        $descclear = preg_replace("/<(\/?meta.*?)>/si", "", $descclear); //过滤meta标签
        $descclear = preg_replace("/<(\/?body.*?)>/si", "", $descclear); //过滤body标签
        $descclear = preg_replace("/<(\/?link.*?)>/si", "", $descclear); //过滤link标签
        $descclear = preg_replace("/<(\/?form.*?)>/si", "", $descclear); //过滤form标签
        $descclear = preg_replace("/cookie/si", "COOKIE", $descclear); //过滤COOKIE标签
        $descclear = preg_replace("/<(applet.*?)>(.*?)<(\/applet.*?)>/si", "", $descclear); //过滤applet标签
        $descclear = preg_replace("/<(\/?applet.*?)>/si", "", $descclear); //过滤applet标签
        $descclear = preg_replace("/<(style.*?)>(.*?)<(\/style.*?)>/si", "", $descclear); //过滤style标签
        $descclear = preg_replace("/<(\/?style.*?)>/si", "", $descclear); //过滤style标签
        $descclear = preg_replace("/<(title.*?)>(.*?)<(\/title.*?)>/si", "", $descclear); //过滤title标签
        $descclear = preg_replace("/<(\/?title.*?)>/si", "", $descclear); //过滤title标签
        $descclear = preg_replace("/<(object.*?)>(.*?)<(\/object.*?)>/si", "", $descclear); //过滤object标签
        $descclear = preg_replace("/<(\/?objec.*?)>/si", "", $descclear); //过滤object标签
        $descclear = preg_replace("/<(noframes.*?)>(.*?)<(\/noframes.*?)>/si", "", $descclear); //过滤noframes标签
        $descclear = preg_replace("/<(\/?noframes.*?)>/si", "", $descclear); //过滤noframes标签
        $descclear = preg_replace("/<(i?frame.*?)>(.*?)<(\/i?frame.*?)>/si", "", $descclear); //过滤frame标签
        $descclear = preg_replace("/<(\/?i?frame.*?)>/si", "", $descclear); //过滤frame标签
        $descclear = preg_replace("/<(script.*?)>(.*?)<(\/script.*?)>/si", "", $descclear); //过滤script标签
        $descclear = preg_replace("/<(\/?script.*?)>/si", "", $descclear); //过滤script标签
        $descclear = preg_replace("/javascript/si", "Javascript", $descclear); //过滤script标签
        $descclear = preg_replace("/vbscript/si", "Vbscript", $descclear); //过滤script标签
        $descclear = preg_replace("/on([a-z]+)\s*=/si", "On\\1=", $descclear); //过滤script标签
        $descclear = preg_replace("/&#/si", "&＃", $descclear); //过滤script标签，如javAsCript:alert();
        //使用正则替换
        $pat = "/<(\/?)(script|i?frame|style|html|body|li|i|map|title|img|link|span|u|font|table|tr|b|marquee|td|strong|div|a|meta|\?|\%)([^>]*?)>/isU";
        $descclear = preg_replace($pat, "", $descclear);
        return $descclear;
    }
}

if (!function_exists("subHtml")) {
    /**
     * 取HTML,并自动补全闭合
     *
     * param $html
     *
     * param $length
     *
     * param $end
     */
    function subHtml($html, $length = 50)
    {
        $result = '';
        $tagStack = array();
        $len = 0;
        $contents = preg_split("~(<[^>]+?>)~si", $html, -1, PREG_SPLIT_NO_EMPTY | PREG_SPLIT_DELIM_CAPTURE);
        foreach ($contents as $tag) {
            if (trim($tag) == "") continue;
            if (preg_match("~<([a-z0-9]+)[^/>]*?/>~si", $tag)) {
                $result .= $tag;
            } else if (preg_match("~</([a-z0-9]+)[^/>]*?>~si", $tag, $match)) {
                if ($tagStack[count($tagStack) - 1] == $match[1]) {
                    array_pop($tagStack);
                    $result .= $tag;
                }
            } else if (preg_match("~<([a-z0-9]+)[^/>]*?>~si", $tag, $match)) {
                array_push($tagStack, $match[1]);
                $result .= $tag;
            } else if (preg_match("~<!--.*?-->~si", $tag)) {
                $result .= $tag;
            } else {
                if ($len + mstrlen($tag) < $length) {
                    $result .= $tag;
                    $len += mstrlen($tag);
                } else {
                    $str = msubstr($tag, 0, $length - $len + 1);
                    $result .= $str;
                    break;
                }
            }
        }
        while (!empty($tagStack)) {
            $result .= '</' . array_pop($tagStack) . '>';
        }
        return $result;
    }
}
if (!function_exists("msubstr")) {

    /**
     * 取中文字符串
     *
     * param $string 字符串
     *
     * param $start 起始位
     *
     * param $length 长度
     *
     * param $charset 编码
     *
     * param $dot 附加字串
     */
    function msubstr($string, $start, $length, $dot = '', $charset = 'UTF-8')
    {
        $string = str_replace(array('&', '"', '<', '>', ' '), array('&', '"', '<', '>', ' '), $string);
        if (strlen($string) <= $length) {
            return $string;
        }
        if (strtolower($charset) == 'utf-8') {
            $n = $tn = $noc = 0;
            while ($n < strlen($string)) {
                $t = ord($string[$n]);
                if ($t == 9 || $t == 10 || (32 <= $t && $t <= 126)) {
                    $tn = 1;
                    $n++;
                } elseif (194 <= $t && $t <= 223) {
                    $tn = 2;
                    $n += 2;
                } elseif (224 <= $t && $t <= 239) {
                    $tn = 3;
                    $n += 3;
                } elseif (240 <= $t && $t <= 247) {
                    $tn = 4;
                    $n += 4;
                } elseif (248 <= $t && $t <= 251) {
                    $tn = 5;
                    $n += 5;
                } elseif ($t == 252 || $t == 253) {
                    $tn = 6;
                    $n += 6;
                } else {
                    $n++;
                }
                $noc++;
                if ($noc >= $length) {
                    break;
                }
            }
            if ($noc > $length) {
                $n -= $tn;
            }
            $strcut = substr($string, 0, $n);
        } else {
            for ($i = 0; $i < $length; $i++) {
                $strcut = "";
                $strcut .= ord($string[$i]) > 127 ? $string[$i] . $string[++$i] : $string[$i];
            }
        }
        return $strcut . $dot;
    }
}

if (!function_exists("mstrlen")) {
    /**
     * 得字符串的长度，包括中英文。
     */
    function mstrlen($str, $charset = 'UTF-8')
    {
        if (function_exists('mb_substr')) {
            $length = mb_strlen($str, $charset);
        } elseif (function_exists('iconv_substr')) {
            $length = iconv_strlen($str, $charset);
        } else {
            preg_match_all("/[\x01-\x7f]|[\xc2-\xdf][\x80-\xbf]|\xe0[\xa0-\xbf][\x80-\xbf]|[\xe1-\xef][\x80-\xbf][\x80-\xbf]|\xf0[\x90-\xbf][\x80-f][\x80-\xbf]|[\xf1-\xf7][\x80-\xbf][\x80-\xbf][\x80-\xbf]/", $str, $ar);
            $length = count($ar[0]);
        }
        return $length;
    }
}

if(!function_exists("get_options")){
    function get_options($name,$default=""){
        $time = 600;
        if(AdminOption::query()->where("name","set_cache_time")->count()){
            if(is_numeric(AdminOption::query()->where("name","set_cache_time")->first()->value)){
                $time = AdminOption::query()->where("name","set_cache_time")->first()->value;
            }
        }
        if($time!=0){
            if(!cache()->has("admin.options.".$name)){
                if(!AdminOption::query()->where("name",$name)->count() or !AdminOption::query()->where("name",$name)->first()->value){
                    return $default;
                    //cache()->set("admin.options.".$name,$default,$time);
                }else{
                    cache()->set("admin.options.".$name,AdminOption::query()->where("name",$name)->first()->value,$time);
                }
            }
            return cache()->get("admin.options.".$name);
        }
        if(!AdminOption::query()->where("name",$name)->count() or !AdminOption::query()->where("name",$name)->first()->value){
            return $default;
            //cache()->set("admin.options.".$name,$default,$time);
        }else{
            return AdminOption::query()->where("name",$name)->first()->value;
        }
    }
}

if(!function_exists("admin_auth")){
    function admin_auth(){
        return new Admin();
    }
}

if(!function_exists("de_stringify")){
    function de_stringify(string $stringify){
        $result = [];
        $data = explode("&",$stringify);
        foreach ($data as $value) {
            $arr = explode("=",$value);
            $result[$arr[0]]=urldecode($arr[1]);
        }
        return $result;
    }
}

if(!function_exists("csrf_token")){
    function csrf_token(){
        if(!session()->has("csrf_token")){
            session()->set("csrf_token",Str::random());
        }
        if(!cache()->has("csrf_token".session()->get("csrf_token"))){
            cache()->set("csrf_token".session()->get("csrf_token"),Str::random(),600);
        }
        return cache()->get("csrf_token".session()->get("csrf_token"));
    }
}

if(!function_exists("recsrf_token")){
    function recsrf_token(){
        return csrf_token();
//        if(!session()->has("csrf_token")){
//            session()->set("csrf_token",Str::random());
//        }
//        cache()->set("csrf_token".session()->get("csrf_token"),Str::random(),300);
//        return cache()->get("csrf_token".session()->get("csrf_token"));
    }
}

if(!function_exists("modifyEnv")){
    function modifyEnv(array $data)
    {
        $envPath = BASE_PATH . '/.env';

        $contentArray = collect(file($envPath, FILE_IGNORE_NEW_LINES));

        $contentArray->transform(function ($item) use ($data) {
            foreach ($data as $key => $value) {
                if (str_contains($item, $key)) {
                    return $key . '=' . $value;
                }
            }

            return $item;
        });

        $content = implode("\n", $contentArray->toArray());

        file_put_contents($envPath, $content);
    }
}

if (!function_exists("Itf_Setting")) {
    function Itf_Setting()
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(SettingInterface::class);
    }
}

if(!function_exists("Router")){
    function Router(){
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(\App\CodeFec\Itf\Route\RouteInterface::class);
    }
}

if(!function_exists("Helpers_Str")){
    function Helpers_Str(): Str
    {
        return new Str();
    }
}

if(!function_exists("Itf")){
    function Itf(){
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(\App\CodeFec\Itf\Itf\ItfInterface::class);
    }
}

if(!function_exists("file_hash")){
    function file_hash($path): string
    {
        if(file_exists(BASE_PATH."/public/".$path)){
            return "/".$path."?version=".md5_file(BASE_PATH."/public/".$path);
        }
        return "/".$path;
    }
}

if(!function_exists("errors")){
    function errors(){
        if(cache()->has("errors")){
            return cache()->get("errors");
        }
        return [];
    }
}

if(!function_exists("url")){
    function url($path=null){
        $url = "http://".env("APP_DOMAIN","请配置APP_DOMAIN");
        if(!$path){
            return $url;
        }
        return $url.$path;
    }
}

if(!function_exists("get_num")){
    function get_num($string): array|string|null
    {
        return preg_replace('/[^0-9]/', '', $string);
    }
}

// 已启动插件列表
if(!function_exists("Plugins_EnList")){
    function Plugins_EnList(){
        if(!cache()->has("admin.plugins.en.list")){
            $arr = [];
            foreach(AdminPlugin::query()->select("name")->get() as $value){
                $arr[]=$value->name;
            }
            cache()->set("admin.plugins.en.list",$arr);
        }
        return cache()->get("admin.plugins.en.list");
    }
}

