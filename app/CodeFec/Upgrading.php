<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec;

use Alchemy\Zippy\Zippy;
use App\Model\AdminOption;
use Swoole\Coroutine\System;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;
use Symfony\Component\Console\Output\OutputInterface;

/**
 * 系统升级.
 */
class Upgrading
{
    public OutputInterface $output;

    public \App\Command\CodeFec\Upgrading $command;

    private string $api_releases = 'https://api.github.com/repos/zhuchunshu/SForum/releases';

    public function __construct(OutputInterface $output, \App\Command\CodeFec\Upgrading $command)
    {
        $this->output = $output;
        $this->command = $command;
    }

    public function get_options($name, $default = '')
    {
        return $this->core_default(@AdminOption::query()->where('name', $name)->first()->value, $default);
    }

    public function core_default($string = null, $default = null)
    {
        if ($string) {
            return $string;
        }
        return $default;
    }

    public function run()
    {
        $url = match ((string) $this->get_options('update_server', '2')) {
            '2' => '',
            '1' => 'https://ghproxy.com/'
        };
        $data = http()->get($this->api_releases);
        $data = $data[0];

        // 获取当前程序版本信息
        $build_info = include BASE_PATH . '/build-info.php';
        $data = array_merge($data, $build_info);
        $version = $data['version'];
        $tag_name = $data['tag_name'];

        // 判断是否不可升级
        if ($tag_name <= $version || $data['prerelease'] === true) {
            $this->command->error('无需升级');
            return;
        }

        // 生成文件下载链接
        $url .= 'https://github.com/zhuchunshu/SForum/archive/' . $tag_name . '.zip';

        // 定义文件存放路径
        $file_path = BASE_PATH . '/runtime/update.zip';

        // 创建下载任务
        $this->download($url, $file_path);
    }

    public function removeFiles(...$values): void
    {
        foreach ($values as $value) {
            \Swoole\Coroutine\System::exec('rm -rf "' . $value . '"');
        }
    }

    private function download(string $download, string $path)
    {
        $this->command->info("开始更新...\n");
        $this->command->info("生成更新锁...\n");
        // 生成更新锁
        file_put_contents(BASE_PATH . '/app/CodeFec/storage/update.lock', time());
        // 备份网站数据
        $this->command->info('开始备份网站数据，网站数据会存放在:' . BASE_PATH . "/runtime/backup/backup.zip 文件中\n");
        backup();
        // 卸载自带组件
        $this->rmPlugins();
        $this->command->info("卸载自带组件...\n");
        // 下载文件
        $this->command->info("\n下载资源包...");
        file_put_contents($path, fopen($download, 'r'));

        // 定义临时压缩包存放目录
        $tmp = BASE_PATH . '/runtime/update';

        // 初始化压缩操作类
        $zippy = Zippy::load();

        // 打开压缩文件
        $archiveTar = $zippy->open($path);

        // 解压
        if (! is_dir($tmp)) {
            mkdir($tmp, 0777);
        }
        // 解压
        $archiveTar->extract($tmp);

        // 获取解压后,插件文件夹的所有目录
        $allDir = allDir($tmp);
        foreach ($allDir as $value) {
            if (file_exists($value . '/CodeFec')) {
                // 替换
                FileUtil()->moveDir($value, BASE_PATH, true);
                // 删除更新锁
                $this->command->info("删除更新锁...\n");
                $this->removeFiles($tmp, $path, BASE_PATH . '/app/CodeFec/storage/update.lock');
                // 清理缓存
                cache()->delete('admin.git.getVersion');
                // 对所有插件进行资源迁移
                $this->command->info("对所有插件进行资源迁移...\n");
                $this->AdminPluginMigrateAll();
                // 更新插件包
                $this->command->info("更新插件包...\n");
                System::exec('php CodeFec CodeFec:PluginsComposerInstall');

                // 重建索引
                $this->command->info("重建索引...\n");
                \Swoole\Coroutine\System::exec('php CodeFec ClearCache');

                $this->command->info('升级完成!');
            }
        }
    }

    private function AdminPluginMigrateAll(): void
    {
        foreach (getEnPlugins() as $name) {
            $plugin_name = $name;

            if (is_dir(plugin_path($plugin_name . '/resources/views')) && ! is_dir(BASE_PATH . '/resources/views/plugins')) {
                \Swoole\Coroutine\System::exec('mkdir ' . BASE_PATH . '/resources/views/plugins');
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

    private function rmPlugins(): void
    {
        $plugins = ['Core', 'User', 'Search', 'Topic', 'Comment', 'Mail'];
        foreach ($plugins as $plugin) {
            $path = plugin_path($plugin);
            $this->removeFiles($path);
        }
        $this->removeFiles(theme_path('CodeFec'));
    }
}
