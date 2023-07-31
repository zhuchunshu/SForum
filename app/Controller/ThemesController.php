<?php

namespace App\Controller;

use Alchemy\Zippy\Zippy;
use App\Middleware\AdminMiddleware;
use App\Model\AdminOption;
use App\Request\Admin\PluginUpload;
use App\Request\Admin\ThemeUpload;
use Hyperf\HttpServer\Annotation\{Controller, GetMapping, Middleware, PostMapping};
use Hyperf\Utils\Str;

#[Controller(prefix: "/admin/themes")]
#[Middleware(AdminMiddleware::class)]
class ThemesController
{

    // 主题管理 - 首页
    #[GetMapping("")]
    public function index()
    {
        return admin_abort('页面不存在');
        return view("admin.themes.index");
    }

    // 主题信息
    #[PostMapping("")]
    public function get()
    {
        return admin_abort('页面不存在');
        return [
            'enable' => get_options("theme", "CodeFec"),
        ];
    }

    // 数据迁移
    #[PostMapping("Migrate")]
    public function Migrate($name = null): array
    {
        return admin_abort('页面不存在');
        if (!$name) {
            if (!request()->input("name")) {
                return Json_Api(403, false, ['msg' => '主题名不能为空']);
            }

            $theme_name = request()->input("name");
        } else {
            $theme_name = $name;
        }

        if (is_dir(theme_path($name . "/resources/views"))) {
            if (!is_dir(BASE_PATH . "/resources/views/themes")) {
                //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/views/themes");
            }
            // if (!is_dir(BASE_PATH . "/resources/views/plugins/" . $plugin_name)) {
            //     //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
            //      \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/views/plugins/" . $plugin_name);
            // }
            // copy_dir(theme_path($plugin_name . "/resources/views"), BASE_PATH . "/resources/views/plugins/" . $plugin_name);
        }
        if (is_dir(theme_path($theme_name . "/resources/assets"))) {
            if (!is_dir(public_path("themes"))) {
                mkdir(public_path("themes"));
            }
            if (!is_dir(public_path("themes/" . $theme_name))) {
                mkdir(public_path("themes/" . $theme_name));
            }
            copy_dir(theme_path($theme_name . "/resources/assets"), public_path("themes/" . $theme_name));
        }
        return Json_Api(200, true, ['msg' => '资源迁移成功!']);
    }

    // 迁移所有资源
    #[PostMapping("MigrateAll")]
    public function MigrateAll(): array
    {
        return admin_abort('页面不存在');
        $this->Migrate(get_options("theme", "CodeFec"));
        return Json_Api(200, true, ['msg' => '资源迁移成功!']);
    }

    // 卸载主题
    #[PostMapping("remove")]
    public function remove()
    {
        return admin_abort('页面不存在');
        $path = theme_path(request()->input('name'));
        if ($path && is_dir($path)) {
            \Swoole\Coroutine\System::exec("rm -rf " . $path);
            if (stripos(system_name(), "Linux") !== false) {
                \Swoole\Coroutine\System::exec("yes | composer du");
            } else {
                \Swoole\Coroutine\System::exec("composer du");
            }
            return Json_Api(200, true, ['msg' => "卸载成功!"]);
        }

        return Json_Api(403, false, ['msg' => "卸载失败,目录:" . $path . " 不存在!"]);
    }

    // 启用主题
    #[PostMapping("enable")]
    public function enable()
    {
        return admin_abort('页面不存在');
        $name = request()->input('name');
        if (theme()->has($name)) {
            $this->setOption([
                'theme' => $name,
            ]);
            return Json_Api(200, true, ['msg' => "主题启用成功!"]);
        }

        return Json_Api(403, false, ['msg' => "启用失败,主题:" . $name . " 不存在!"]);
    }

    private function setOption($data = []): void
    {
//        return admin_abort('页面不存在');
        foreach ($data as $key => $value) {
            if (AdminOption::query()->where("name", $key)->exists()) {
                AdminOption::query()->where("name", $key)->update(['value' => $value]);
            } else {
                AdminOption::query()->create(['name' => $key, 'value' => $value]);
            }
        }
        options_clear();
    }

    // 上传主题
    #[GetMapping("upload")]
    public function upload()
    {
        return admin_abort('页面不存在');
        return view("admin.themes.upload");
    }

    // 上传主题
    #[PostMapping("upload")]
    public function upload_submit(ThemeUpload $request)
    {
        return admin_abort('页面不存在');
        // 不带后缀的文件名
        $filename = Str::before($request->file('file')->getClientFilename(), '.');
        // 带后缀的文件名
        $getClientFilename = $request->file('file')->getClientFilename();

        // 移动文件
        $request->file('file')->moveTo(theme_path($request->file('file')->getClientFilename()));

        // 初始化压缩操作类
        $zippy = Zippy::load();

        // 打开压缩文件
        $archiveTar = $zippy->open(theme_path($getClientFilename));

        // 解压
        if (!is_dir(theme_path($filename))) {
            mkdir(theme_path($filename), 0777);
        }

        $archiveTar->extract(theme_path($filename));

        // 获取解压后,插件文件夹的所有目录
        $allDir = allDir(theme_path($filename));
        foreach ($allDir as $value) {
            if (file_exists($value . "/.dirName")) {
                $dirname = file_get_contents($value . "/.dirName");
                if (!$dirname) {
                    $this->removeFiles(theme_path($getClientFilename), theme_path($filename));
                    return redirect()->with('danger', '.dirName文件为空')->url('/admin/themes/upload')->go();
                }
                FileUtil()->moveDir($value, theme_path($dirname), true);
                $this->removeFiles(theme_path($getClientFilename));
                return redirect()->with('success', '主题上传成功!')->url('/admin/themes/upload')->go();
            }
        }
        $this->removeFiles(theme_path($getClientFilename), theme_path($filename));
        return redirect()->with('danger', '主题安装失败,没有找到 .dirName 文件')->url('/admin/themes/upload')->go();
    }

    public function removeFiles(...$values): void
    {
        foreach ($values as $value) {
            \Swoole\Coroutine\System::exec('rm -rf "' . $value . '"');
        }
    }
}