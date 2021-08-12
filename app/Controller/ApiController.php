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

namespace App\Controller;

use App\Model\AdminPlugin;
use Hyperf\DbConnection\Db;
use Illuminate\Support\Arr;
use Illuminate\Support\Str;
use App\Middleware\AdminMiddleware;
use League\Flysystem\FileExistsException;
use Hyperf\HttpServer\Annotation\Middleware;
use Psr\SimpleCache\InvalidArgumentException;
use Symfony\Component\Console\Input\ArrayInput;
use Hyperf\HttpServer\Annotation\AutoController;
use Hyperf\HttpServer\Contract\RequestInterface;
use Symfony\Component\Console\Output\NullOutput;

/**
 * @AutoController
 * Class ApiController
 */
class ApiController
{
    public function avatar(RequestInterface $request): array
    {
        $email = $request->input('email');
        return Json_Api(200, true, ['avatar' => 'https://dn-qiniu-avatar.qbox.me/avatar/' . md5($email)]);
    }

    /**
     * @Middleware(AdminMiddleware::class)
     */
    public function menu(): array
    {
        return Json_Api(200, true, menu()->get());
    }

    /**
     * @Middleware(AdminMiddleware::class)
     */
    public function AdminPluginList(): array
    {
        $array = AdminPlugin::query()->where("status", 1)->get();
        $result = [];
        foreach ($array as $value) {
            $result[] = $value->name;
        }
        return Json_Api(200, true, ['data' => $result]);
    }

    /**
     * @Middleware(AdminMiddleware::class)
     */
    public function AdminPluginSave(): array
    {

        Db::table('admin_plugins')->truncate();
        if (request()->input("data") && is_array(request()->input("data"))) {
            $data = request()->input("data");
            $arr = [];
            foreach ($data as $value) {
                $arr[] = ['name' => $value, 'status' => 1, 'created_at' => date("Y-m-d H:i:s"), 'updated_at' => date("Y-m-d H:i:s")];
            }
            Db::table('admin_plugins')->insert($arr);
        }
        $myfile = fopen(BASE_PATH . "/app/CodeFec/storage/logs_plugins.php", "w") or die("Unable to open file!");
        $txt = "- " . date("Y-m-d H:i:s") . "插件状态变动:\n" . json_encode(request()->input('data'));
        fwrite($myfile, $txt);
        fclose($myfile);
        try {
            cache()->delete("admin.plugins.en.list");
        } catch (InvalidArgumentException $e) {
        }
        return Json_Api(200, true, ['msg' => "更新成功!"]);
    }

    public function AdminPluginMigrate($name=null): array
    {
        if(!$name){
            if (!request()->input("name")) {
                return Json_Api(403, false, ['msg' => '插件名不能为空']);
            }

            $plugin_name = request()->input("name");
        }else{
            $plugin_name = $name;
        }

        if (is_dir(plugin_path($plugin_name . "/resources/views"))) {
            if (!is_dir(BASE_PATH . "/resources/views/plugins")) {
                //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                exec("mkdir " . BASE_PATH . "/resources/views/plugins");
            }
            if (!is_dir(BASE_PATH . "/resources/views/plugins/" . $plugin_name)) {
                //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                exec("mkdir " . BASE_PATH . "/resources/views/plugins/" . $plugin_name);
            }
            copy_dir(plugin_path($plugin_name . "/resources/views"), BASE_PATH . "/resources/views/plugins/" . $plugin_name);
        }
        if (is_dir(plugin_path($plugin_name . "/resources/assets"))) {
            if (!is_dir(public_path("plugins"))) {
                mkdir(public_path("plugins"));
            }
            if (!is_dir(public_path("plugins/" . $plugin_name))) {
                mkdir(public_path("plugins/" . $plugin_name));
            }
            copy_dir(plugin_path($plugin_name . "/resources/assets"), public_path("plugins/" . $plugin_name));
        }
        if (is_dir(plugin_path($plugin_name . "/src/migrations"))) {
            $params = ["command" => "CodeFec:migrate", "path" => plugin_path($plugin_name . "/src/migrations")];

            $input = new ArrayInput($params);
            $output = new NullOutput();

            $container = \Hyperf\Utils\ApplicationContext::getContainer();

            /** @var Application $application */
            $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
            $application->setAutoExit(false);

            // 这种方式: 不会暴露出命令执行中的异常, 不会阻止程序返回
            $exitCode = $application->run($input, $output);
        }
        return Json_Api(200, true, ['msg' => '资源迁移成功!']);
    }

    public function AdminPluginMigrateAll(): array
    {
        foreach (Plugins_EnList() as $name){
            $this->AdminPluginMigrate($name);
        }
        return Json_Api(200, true, ['msg' => '资源迁移成功!']);
    }

    /**
     * @Middleware(AdminMiddleware::class)
     */
    public function AdminPluginRemove(): array
    {
        if (request()->input("path")) {
            exec("rm -rf " . request()->input("path"), $result, $status);
            return Json_Api(200, true, ['msg' => "卸载成功!"]);
        } else {
            return Json_Api(403, false, ['msg' => "卸载失败,目录:" . request()->input("path") . " 不存在!"]);
        }
    }

    public function AdminErrorRedirect(): array
    {
        $list = [
            "/admin" => "/admin/login",
            "/admin/login" => "/admin"
        ];
        if (request()->input("path", null)) {
            $path = request()->input("path", null);
            if (Arr::has($list, $path)) {
                return Json_Api(200, true, ["data" => $list[$path]]);
            } else {
                return Json_Api(403, false, ["data" => "#"]);
            }
        } else {
            return Json_Api(403, false, ["data" => "#"]);
        }
    }
}
