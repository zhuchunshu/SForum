<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use Alchemy\Zippy\Zippy;
use App\CodeFec\Admin\Admin;
use App\CodeFec\Itf\Setting\SettingInterface;
use App\CodeFec\Menu\MenuInterface;
use App\CodeFec\Plugins;
use App\CodeFec\View\Beautify_Html;
use App\Model\AdminOption;
use Hyperf\Context\Context;
use Hyperf\Contract\SessionInterface;
use Hyperf\Contract\StdoutLoggerInterface;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Hyperf\HttpServer\Response;
use Hyperf\Logger\LoggerFactory;
use Hyperf\Paginator\UrlWindow;
use Hyperf\Server\ServerFactory;
use Hyperf\Utils\{ApplicationContext};
use Hyperf\View\RenderInterface;
use Hyperf\ViewEngine\Contract\FactoryInterface;
use Illuminate\Support\Arr;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;
use Overtrue\Http\Client;
use Psr\Container\ContainerInterface;
use Psr\EventDispatcher\EventDispatcherInterface;
use Psr\Http\Message\ServerRequestInterface;
use Swoole\Coroutine\System;

function public_path($path = ''): string
{
    if ($path !== '') {
        return config('server.settings.document_root') . '/' . ltrim($path, '/');
    }
    return config('server.settings.document_root');
}

if (! function_exists('mix_manifest')) {
    function mix_manifest()
    {
        return file_get_contents(public_path('mix-manifest.json'));
    }
}

if (! function_exists('mix')) {
    function mix($path)
    {
//        $list = mix_manifest();
//        $result = json_decode($list, true);
//        if (Arr::has($result, '/' . $path)) {
//            return $result['/' . $path];
//        }
//        return '/' . $path;
        return file_hash($path);
    }
}

if (! function_exists('arr_has')) {
    function arr_has($array, $keys): bool
    {
        return Arr::has($array, $keys);
    }
}

/*
 * 容器实例
 */
if (! function_exists('container')) {
    function container(): ContainerInterface
    {
        return ApplicationContext::getContainer();
    }
}

/*
 * redis 客户端实例
 */
if (! function_exists('redis')) {
    function redis()
    {
        return container()->get(Redis::class);
    }
}

/*
 * server 实例 基于 swoole server
 */
if (! function_exists('server')) {
    function server()
    {
        return container()->get(ServerFactory::class)->getServer()->getServer();
    }
}

/*
 * 缓存实例 简单的缓存
 */
if (! function_exists('cache')) {
    function cache()
    {
        return container()->get(Psr\SimpleCache\CacheInterface::class);
    }
}

/*
 * 控制台日志
 */
if (! function_exists('stdLog')) {
    function stdLog()
    {
        return container()->get(StdoutLoggerInterface::class);
    }
}

/*
 * 文件日志
 */
if (! function_exists('logger')) {
    function logger()
    {
        return container()->get(LoggerFactory::class)->make();
    }
}

if (! function_exists('response')) {
    function response()
    {
        return container()->get(ResponseInterface::class);
    }
}

if (! function_exists('PsrResponse')) {
    function PsrResponse()
    {
        return container()->get(\Psr\Http\Message\ResponseInterface::class);
    }
}

if (! function_exists('ResponseObj')) {
    function ResponseObj(): Response
    {
        return new Response();
    }
}

if (! function_exists('SwooleStream')) {
    function SwooleStream($contents): SwooleStream
    {
        return new SwooleStream($contents);
    }
}

if (! function_exists('request')) {
    function request(): Hyperf\HttpServer\Request
    {
        return new Hyperf\HttpServer\Request();
    }
}

if (! function_exists('path_class')) {
    function path_class()
    {
        $path = request()->path();
        $result = str_replace('/', '-', $path);
        $result = Str::before($result, '.');
        if ($result == '-') {
            return 'main';
        }
        return $result;
    }
}

if (! function_exists('menu')) {
    function menu()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(MenuInterface::class);
    }
}

if (! function_exists('view')) {
    function view(string $view, array $data = [], int $code = 200)
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(RenderInterface::class)->render($view, $data, $code);
        if (env('APP_ENV') === 'dev') {
            return $result;
        }
        $body = minify_html((string) $result->getBody());
        return $container->get(RenderInterface::class)->renderR($body, $code);
    }
}

if (! function_exists('menu_pd')) {
    function menu_pd($id)
    {
        $i = 0;
        foreach (menu()->get() as $key => $value) {
            if (arr_has($value, 'parent_id')) {
                if ($value['parent_id'] === $id) {
                    ++$i;
                }
            }
        }
        return $i;
    }
}

if (! function_exists('menu_pdArr')) {
    function menu_pdArr($id)
    {
        $arr = [];
        foreach (menu()->get() as $key => $value) {
            if (arr_has($value, 'parent_id')) {
                if ($value['parent_id'] == $id) {
                    $arr[$key] = $value;
                }
            }
        }
        return $arr;
    }
}

if (! function_exists('Json_Api')) {
    function Json_Api(int $code = 200, bool $success = true, object | array | string $result = []): array
    {
        return [
            'code' => $code,
            'success' => $success,
            'result' => $result,
            'RequestTime' => date('Y-m-d H:i:s'),
        ];
    }
}

if (! function_exists('json_api')) {
    function json_api(int $code = 200, bool $success = true, object | array | string $result = []): array
    {
        return [
            'code' => $code,
            'success' => $success,
            'result' => $result,
            'RequestTime' => date('Y-m-d H:i:s'),
        ];
    }
}

if (! function_exists('session')) {
    function session()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(SessionInterface::class);
    }
}

// 获取目录下的所有文件
if (! function_exists('getPath')) {
    function getPath($path)
    {
        if (! is_dir($path)) {
            return false;
        }
        $arr = [];
        $data = scandir($path);
        foreach ($data as $value) {
            if ($value != '.' && $value != '..') {
                $arr[] = $value;
            }
        }
        return $arr;
    }
}

if (! function_exists('getPathDir')) {
    function getPathDir($path)
    {
        if (! is_dir($path)) {
            return false;
        }
        $arr = [];
        $data = scandir($path);
        foreach ($data as $value) {
            if ($value != '.' && $value != '..' && is_dir($path . '/' . $value)) {
                $arr[] = $value;
            }
        }
        return $arr;
    }
}

if (! function_exists('plugin_path')) {
    function plugin_path($path = null): string
    {
        if (! $path) {
            return BASE_PATH . '/app/Plugins';
        }
        return BASE_PATH . '/app/Plugins/' . $path;
    }
}

if (! function_exists('theme_path')) {
    function theme_path($path = null): string
    {
        if (! $path) {
            return BASE_PATH . '/app/Themes';
        }
        return BASE_PATH . '/app/Themes/' . $path;
    }
}

if (! function_exists('lang_path')) {
    function lang_path($path = null): string
    {
        if (! $path) {
            return BASE_PATH . '/app/Languages';
        }
        return BASE_PATH . '/app/Languages/' . $path;
    }
}

if (! function_exists('read_file')) {
    function read_file($file_path): ?string
    {
        if (file_exists($file_path)) {
            return File::get($file_path);
        }

        return null;
    }
}

if (! function_exists('admin_abort')) {
    /**
     * @param array|string $data
     * @return \Psr\Http\Message\ResponseInterface
     */
    function admin_abort(array | string $data, int $code = 403, string $redirect = null): Psr\Http\Message\ResponseInterface
    {
        if (is_string($data)) {
            $array = ['msg' => $data];
        } else {
            $array = $data;
        }
        if (request()->isMethod('POST') || request()->input('data') === 'json') {
            return response()->json(Json_Api($code, false, $array));
        }
        $_redirect = @request()->getHeader('referer')[0] ?: '/';
        $redirect = $redirect ?: $_redirect;
        return view('admin.error', ['redirect' => $redirect, 'code' => $code, 'data' => $array], $code);
    }
}

if (! function_exists('get_plugins_doc')) {
    function get_plugins_doc($class): array
    {
        $re = new ReflectionClass(new $class());
        $content = $re->getDocComment();
        $preg = '/@+(.*)/';
        preg_match_all($preg, $content, $result);
        $result = $result[1];
        $arr = [];
        foreach ($result as $key => $value) {
            $result1 = explode(' ', $value);
            $arr[$result1[0]] = $result1[1];
        }
        $data = [];
        foreach ($arr as $key => $value) {
            if ($key === 'package') {
                $data['description'] = $value;
            } else {
                $data[$key] = $value;
            }
        }
        return $data;
    }
}

if (! function_exists('deldir')) {
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
                    if ($val != '.' && $val != '..') {
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

if (! function_exists('copy_dir')) {
    function copy_dir($src, $dst)
    {
        $dir = opendir($src);
        @mkdir($dst);
        while (false !== ($file = readdir($dir))) {
            if (($file != '.') && ($file != '..')) {
                if (is_dir($src . '/' . $file)) {
                    copy_dir($src . '/' . $file, $dst . '/' . $file);
                    continue;
                }

                copy($src . '/' . $file, $dst . '/' . $file);
            }
        }
        closedir($dir);
    }
}

if (! function_exists('verify_ip')) {
    function verify_ip($realip)
    {
        return filter_var($realip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4);
    }
}

if (! function_exists('get_client_ip')) {
    function get_client_ip()
    {
        /**
         * @var ServerRequestInterface $request
         */
        $request = Context::get(ServerRequestInterface::class);
        $ip_addr = @explode(',', $request->getHeaderLine('x-forwarded-for'))[0];
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        $ip_addr = @explode(',', $request->getHeaderLine('remote-host'))[0];
        if (verify_ip($ip_addr)) {
            return $ip_addr;
        }
        $ip_addr = @explode(',', $request->getHeaderLine('x-real-ip'))[0];
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

if (! function_exists('make_page')) {
    function make_page($page, $default = 'default'): Psr\Http\Message\ResponseInterface
    {
        $window = UrlWindow::make($page);

        $elements = array_filter([
            $window['first'],
            is_array($window['slider']) ? '...' : null,
            $window['slider'],
            is_array($window['last']) ? '...' : null,
            $window['last'],
        ]);
        return view('shared.page.' . $default, ['paginator' => $page, 'elements' => $elements]);
    }
}

if (! function_exists('get_options')) {
    function get_options($name, $default = '')
    {
        if (! cache()->has('admin.options.' . $name)) {
            cache()->set('admin.options.' . $name, @AdminOption::query()->where('name', $name)->first()->value);
        }
        return core_default(cache()->get('admin.options.' . $name), $default);
    }
}

if (! function_exists('set_options')) {
    function set_options($name, $value): void
    {
        if (AdminOption::query()->where('name', $name)->exists()) {
            AdminOption::query()->where('name', $name)->update(['value' => $value]);
        } else {
            AdminOption::query()->create(['name' => $name, 'value' => $value]);
        }
        cache()->set('admin.options.' . $name, $value);
        options_clear();
    }
}

if (! function_exists('get_options_nocache')) {
    function get_options_nocache($name, $default = '')
    {
        if (! AdminOption::query()->where('name', $name)->exists() || ! AdminOption::query()->where('name', $name)->first()->value) {
            return $default;
        }

        return AdminOption::query()->where('name', $name)->first()->value;
    }
}

if (! function_exists('options_clear')) {
    function options_clear()
    {
        foreach (AdminOption::query()->get() as $value) {
            cache()->delete('admin.options.' . $value->name);
        }
    }
}

if (! function_exists('admin_auth')) {
    function admin_auth(): Admin
    {
        return new Admin();
    }
}

if (! function_exists('de_stringify')) {
    function de_stringify(string $stringify): array
    {
        $result = [];
        $data = explode('&', $stringify);
        foreach ($data as $value) {
            $arr = explode('=', $value);
            $result[$arr[0]] = urldecode($arr[1]);
        }
        return $result;
    }
}

if (! function_exists('csrf_token')) {
    function csrf_token()
    {
        if (! session()->has('CSRF_TOKEN')) {
            session()->set('CSRF_TOKEN', Str::random());
        }
        if (! cache()->has('CSRF_TOKEN' . session()->get('CSRF_TOKEN'))) {
            $k = \Hyperf\Utils\Str::random(25);
            cache()->set('CSRF_TOKEN' . session()->get('CSRF_TOKEN'), $k);
        }
        return cache()->get('CSRF_TOKEN' . session()->get('CSRF_TOKEN'));
    }
}

if (! function_exists('recsrf_token')) {
    function recsrf_token()
    {
        if (! session()->has('CSRF_TOKEN')) {
            session()->set('CSRF_TOKEN', Str::random());
        }
        $k = \Hyperf\Utils\Str::random(25);
        cache()->set('CSRF_TOKEN' . session()->get('CSRF_TOKEN'), $k);
        return cache()->get('CSRF_TOKEN' . session()->get('CSRF_TOKEN'));
    }
}

if (! function_exists('modifyEnv')) {
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

if (! function_exists('Itf_Setting')) {
    function Itf_Setting()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(SettingInterface::class);
    }
}

if (! function_exists('Router')) {
    function Router()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(\App\CodeFec\Itf\Route\RouteInterface::class);
    }
}

if (! function_exists('Themes')) {
    function Themes()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(\App\CodeFec\Itf\Theme\ThemeInterface::class);
    }
}

if (! function_exists('Theme')) {
    function Theme(): App\CodeFec\Themes
    {
        return new \App\CodeFec\Themes();
    }
}

if (! function_exists('Helpers_Str')) {
    function Helpers_Str(): Str
    {
        return new Str();
    }
}

if (! function_exists('Itf')) {
    function Itf()
    {
        return \Hyperf\Utils\ApplicationContext::getContainer()->get(\App\CodeFec\Itf\Itf\ItfInterface::class);
    }
}

if (! function_exists('file_hash')) {
    function file_hash($path): string
    {
        if (file_exists(BASE_PATH . '/public/' . $path)) {
            return '/' . $path . '?version=' . build_info()->version;
        }
        return '/' . $path;
    }
}

if (! function_exists('errors')) {
    function errors()
    {
        if (cache()->has('errors')) {
            return cache()->get('errors');
        }
        return [];
    }
}

if (! function_exists('url')) {
    function url($path = null)
    {
        $url = get_options('APP_URL', null);
        $url = $url ?: \App\Helpers\Url::getReqFullHost(request());
        if (! $path) {
            return $url;
        }
        return $url . $path;
    }
}

if (! function_exists('url_source')) {
    function url_source($path = null)
    {
        $url = 'http://' . request()->getHeader('host')[0];
        if (! $path) {
            return $url;
        }
        return $url . $path;
    }
}

if (! function_exists('ws_url')) {
    function ws_url($path = null)
    {
        $url = get_options('APP_WS_URL');
        if (! $path) {
            return $url;
        }
        return $url . $path;
    }
}

if (! function_exists('get_num')) {
    function get_num($string): array | string | null
    {
        return preg_replace('/[^0-9]/', '', $string);
    }
}

// 已启动插件列表
if (! function_exists('getEnPlugins')) {
    function getEnPlugins()
    {
        return (new Plugins())->getEnPlugins();
    }
}

// 已启动插件列表
if (! function_exists('plugins')) {
    function plugins(): Plugins
    {
        return new Plugins();
    }
}

if (! function_exists('http')) {
    function http($response_type = 'array'): Client
    {
        return Client::create([
            'response_type' => $response_type,
            'verify' => false,
        ]);
    }
}

if (! function_exists('EventDispatcher')) {
    function EventDispatcher()
    {
        return container()->get(EventDispatcherInterface::class);
    }
}

if (! function_exists('captcha')) {
    function captcha(): App\CodeFec\Captcha
    {
        return new \App\CodeFec\Captcha();
    }
}

if (! function_exists('fileUtil')) {
    function fileUtil(): App\CodeFec\FileUtil
    {
        return new \App\CodeFec\FileUtil();
    }
}

if (! function_exists('allDir')) {
    function allDir($dir)
    { //遍历目录下的文件夹
        $data = scandir($dir);
        $arr = [];
        // 把自身写进去
        $arr[] = $dir;

        foreach ($data as $value) {
            if ($value !== '.' && $value !== '..' && is_dir($dir . '/' . $value)) {
                foreach (allDir($dir . '/' . $value) as $v) {
                    $arr[] = $v;
                }
            }
        }
        return $arr;
    }
}

// --压缩-- 美化html
function minify_html($html): array | string | null
{
    $beautify = new Beautify_Html([
        'indent_inner_html' => false,
        'indent_char' => ' ',
        'indent_size' => 2,
        'wrap_line_length' => 32786,
        'unformatted' => ['code', 'pre', 'span'],
        'preserve_newlines' => false,
        'max_preserve_newlines' => 32786,
        'indent_scripts' => 'keep', // keep|separate|normal
    ]);
    return $beautify->beautify($html);
}

if (! function_exists('build_info')) {
    function build_info()
    {
        $data = include BASE_PATH . '/build-info.php';
        return json_decode(json_encode($data));
    }
}

// 获取系统名
if (! function_exists('system_name')) {
    function system_name(): bool | string | null
    {
        return str_replace("\n", '', shell_exec('echo $(uname)'));
    }
}
if (! function_exists('cmd_which')) {
    function cmd_which($bin): bool | string | null
    {
        $cmd = shell_exec('which ' . $bin);
        if ($cmd) {
            return str_replace("\n", '', $cmd);
        }
        return false;
    }
}

if (! function_exists('get_user_agent')) {
    /**
     * 获取客户端user agent信息.
     * @return mixed|string
     */
    function get_user_agent()
    {
        return request()->getHeader('user-agent')[0];
    }
}

if (! function_exists('get_client_ip_data')) {
    /**
     * 获取ip 信息.
     * @param null $ip
     * @throws \Gai871013\IpLocation\Exceptions\InvalidArgumentException
     */
    function get_client_ip_data($ip = null): array
    {
        $result = (new \Gai871013\IpLocation\IpLocation())->getLocation($ip);
        if (Arr::has($result, 'country')) {
            $result['pro'] = $result['country'];
        } else {
            $result['pro'] = '';
        }
        $result['country'] = intercept_province($result['country']);
        $result['pro'] = intercept_province($result['pro']);
        return $result;
    }
}

if (! function_exists('language')) {
    function language(): App\CodeFec\Language
    {
        return new \App\CodeFec\Language();
    }
}

if (! function_exists('remove_bbCode')) {
    function remove_bbCode($content)
    {
        $pattern = '/\\[(.*?)\\](.*?)\\[\\/(.*?)\\]/is';

        $content = preg_replace_callback($pattern, function ($match) {
            return '';
        }, $content);
        return trim($content ?: ' ');
    }
}

if (! function_exists('content_brief')) {
    function content_brief($content, string | int $len = 100): string
    {
        if (@! $content) {
            return $content;
        }
        $len = (int) $len;
        // hook post_brief_start.php
        $content = strip_tags($content);
        $content = htmlspecialchars($content);
        $content = remove_bbCode($content) ?: '';
        $content = \Hyperf\Utils\Str::limit($content, $len);
        return htmlspecialchars_decode($content, ENT_QUOTES);
    }
}

if (! function_exists('admin_log')) {
    function admin_log(): App\CodeFec\Admin\LogServer
    {
        return new \App\CodeFec\Admin\LogServer();
    }
}

if (! function_exists('pay')) {
    /**
     * 支付服务
     * @return \App\Plugins\Core\src\Lib\Pay\PayService
     */
    function pay(): App\Plugins\Core\src\Lib\Pay\PayService
    {
        return new \App\Plugins\Core\src\Lib\Pay\PayService();
    }
}

if (! function_exists('qr_code')) {
    function qr_code(): SimpleSoftwareIO\QrCode\Generator
    {
        return new \SimpleSoftwareIO\QrCode\Generator();
    }
}

if (! function_exists('backup')) {
    /**
     * 备份网站数据.
     * @param null|mixed $filename backup压缩文件 文件名
     */
    function backup(mixed $filename = null): string
    {
        if (! is_dir(BASE_PATH . '/runtime/backup')) {
            System::exec('cd ' . BASE_PATH . '/runtime' . '&& mkdir ' . 'backup');
        }
        if (! $filename) {
            $filename = BASE_PATH . '/runtime/backup/backup.zip';
        } else {
            $filename = BASE_PATH . '/runtime/backup/' . $filename . '.zip';
        }
        _menu_instance()->backup();
        $sql_backup_name = null;
        if (cmd_which('mysqldump')) {
            $sql_backup_name = Str::random(40) . '.sql';
            $sql_backup_name = BASE_PATH . '/runtime/backup/' . $sql_backup_name;
            System::exec('mysqldump -u ' . config('databases.default.username') . ' -p' . config('databases.default.password') . ' ' . config('databases.default.database') . ' > "' . $sql_backup_name . '"');
        }
        $backup_files = [
            BASE_PATH . '/app',
            'menu' => BASE_PATH . '/runtime/backup/menu',
            public_path(),
            BASE_PATH . '/.env',
            BASE_PATH . '/composer.json',
            BASE_PATH . '/composer.lock',
        ];
        if ($sql_backup_name && file_exists($sql_backup_name)) {
            $backup_files['backup.sql'] = $sql_backup_name;
        }
        $zippy = Zippy::load();
        $zippy->create($filename, $backup_files, true);
        System::exec('rm -rf "' . $sql_backup_name . '"');
        return $filename;
    }
}

if (! function_exists('system_clear_cache')) {
    function system_clear_cache()
    {
        removeFiles(BASE_PATH . '/runtime/container', BASE_PATH . '/runtime/view');
        \Swoole\Coroutine\System::exec('php CodeFec CodeFec:view-engine-cache && php CodeFec');
    }
}

if (! function_exists('removeFiles')) {
    function removeFiles(...$values): void
    {
        foreach ($values as $value) {
            \Swoole\Coroutine\System::exec('rm -rf "' . $value . '"');
        }
    }
}

if (! function_exists('_menu')) {
    function _menu()
    {
        return call_user_func([new \App\Plugins\Core\Menu(), 'get']);
    }
}

if (! function_exists('_menu_keys')) {
    function _menu_keys()
    {
        return call_user_func([new \App\Plugins\Core\Menu(), 'get_keys']);
    }
}

if (! function_exists('_menu_data')) {
    /**
     * @param int|string $id menu id
     * @return mixed
     */
    function _menu_get_data(int | string $id)
    {
        return call_user_func([new \App\Plugins\Core\Menu(), 'get_data'], $id);
    }
}

if (! function_exists('_menu_instance')) {
    function _menu_instance(): App\Plugins\Core\Menu
    {
        return new \App\Plugins\Core\Menu();
    }
}
if (! function_exists('get_component_view_name')) {
    function get_component_view_name($name): string
    {
        $view = 'customize.component.' . $name;
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        $factory = $container->get(FactoryInterface::class);
        if (! $factory->exists($view)) {
            return 'shared.viewIsNull';
        }
        return $view;
    }
}

if (! function_exists('make_template')) {
    function make_template(string $file, array $data)
    {
        $template = get_file_content($file);

        foreach ($data as $key => $value) {
            $template = str_replace('{{' . $key . '}}', $value, $template);
        }

        return $template;
    }
}

if (! function_exists('get_file_content')) {
    function get_file_content($file): bool | string
    {
        $handle = fopen($file, 'r');
        $content = fread($handle, filesize($file));
        fclose($handle);
        return $content;
    }
}

// 截取内容摘要
if (! function_exists('get_content_brief')) {
    function get_content_brief($content, $len = 200): string
    {
        if (@! $content) {
            return $content;
        }
        $len = (int) $len;
        // hook post_brief_start.php
        $content = strip_tags($content);
        $content = htmlspecialchars($content);
        $content = remove_bbCode($content) ?: '';
        $content = \Hyperf\Utils\Str::limit($content, $len);
        $content = trim(preg_replace('/\s+/', ' ', $content));
        return htmlspecialchars_decode($content, ENT_QUOTES);
    }
}