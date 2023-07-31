<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Controller;

use Alchemy\Zippy\Zippy;
use App\Request\Admin\PluginUpload;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Hyperf\Utils\Str;
use Psr\Container\ContainerExceptionInterface;
use Psr\Container\NotFoundExceptionInterface;
use Psr\Http\Message\ResponseInterface;
use Swoole\Coroutine\System;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;

#[Controller(prefix: '/admin/plugins')]
#[Middleware(\App\Middleware\AdminMiddleware::class)]
class PluginsController
{
    #[GetMapping('')]
    public function index(): ResponseInterface
    {
        return view('admin.plugins.index', ['page' => $this->page()]);
    }

    // 上传插件

    #[GetMapping('upload')]
    public function upload()
    {
        return view('admin.plugins.upload');
    }

    #[GetMapping('logo')]
    public function logo()
    {
        $plugin = request()->input('plugin');
        if (! file_exists(BASE_PATH . '/app/Plugins/' . $plugin . '/' . $plugin . '.png')) {
            return admin_abort('图片不存在', 404);
        }
        $file = fread(fopen(BASE_PATH . '/app/Plugins/' . $plugin . '/' . $plugin . '.png', 'rb'), filesize(BASE_PATH . '/app/Plugins/' . $plugin . '/' . $plugin . '.png'));
        return response()->raw($file);
    }

    // 上传插件

    #[PostMapping('upload')]
    public function upload_submit(PluginUpload $request)
    {
        // 不带后缀的文件名
        $filename = Str::before($request->file('file')->getClientFilename(), '.');
        // 带后缀的文件名
        $getClientFilename = $request->file('file')->getClientFilename();

        // 移动文件
        $request->file('file')->moveTo(plugin_path($request->file('file')->getClientFilename()));

        // 初始化压缩操作类
        $zippy = Zippy::load();

        // 打开压缩文件
        $archiveTar = $zippy->open(plugin_path($getClientFilename));

        // 解压
        if (! is_dir(plugin_path($filename))) {
            mkdir(plugin_path($filename), 0777);
        }

        $archiveTar->extract(plugin_path($filename));

        // 获取解压后,插件文件夹的所有目录
        $allDir = allDir(plugin_path($filename));
        foreach ($allDir as $value) {
            if (file_exists($value . '/.dirName')) {
                $dirname = file_get_contents($value . '/.dirName');
                if (! $dirname) {
                    $this->removeFiles(plugin_path($getClientFilename), plugin_path($filename));
                    return redirect()->with('danger', '.dirName文件为空')->url('/admin/plugins/upload')->go();
                }
                $this->removeFiles(plugin_path($dirname));
                FileUtil()->moveDir($value, plugin_path($dirname), true);
                $this->removeFiles(plugin_path($getClientFilename), plugin_path($filename));
                $this->migrate_resources($dirname);
                return redirect()->with('success', '插件上传成功!')->url('/admin/plugins')->go();
            }
        }

        $this->removeFiles(plugin_path($getClientFilename), plugin_path($filename));
        System::exec('php CodeFec ClearCache');
        return redirect()->with('danger', '插件安装失败,没有找到 .dirName 文件')->url('/admin/plugins/upload')->go();
    }

    public function removeFiles(...$values): void
    {
        foreach ($values as $value) {
            \Swoole\Coroutine\System::exec('rm -rf "' . $value . '"');
        }
    }

    private function page()
    {
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection(plugins()->get_all_data());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count(plugins()->get_all_data()), $perPage, $currentPage);
    }

    /**
     * 迁移资源.
     * @param mixed $plugin_name 插件目录名
     * @throws ContainerExceptionInterface
     * @throws NotFoundExceptionInterface
     */
    private function migrate_resources($plugin_name)
    {
        if (is_dir(plugin_path($plugin_name . '/resources/views'))) {
            if (! is_dir(BASE_PATH . '/resources/views/plugins')) {
                //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                \Swoole\Coroutine\System::exec('mkdir ' . BASE_PATH . '/resources/views/plugins');
            }
        }
        if (is_dir(plugin_path($plugin_name . '/resources/assets'))) {
            if (! is_dir(public_path('plugins'))) {
                mkdir(public_path('plugins'));
            }
            if (! is_dir(public_path('plugins/' . $plugin_name))) {
                mkdir(public_path('plugins/' . $plugin_name));
            }
            copy_dir(plugin_path($plugin_name . '/resources/assets'), public_path('plugins/' . $plugin_name));
        }
        if (is_dir(plugin_path($plugin_name . '/src/migrations'))) {
            $params = ['command' => 'CodeFec:migrate', 'path' => plugin_path($plugin_name . '/src/migrations')];

            $input = new ArrayInput($params);
            $output = new NullOutput();

            $container = \Hyperf\Utils\ApplicationContext::getContainer();

            /** @var Application $application */
            $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
            $application->setAutoExit(false);

            // 这种方式: 不会暴露出命令执行中的异常, 不会阻止程序返回
            $exitCode = $application->run($input, $output);
        }
    }
}
