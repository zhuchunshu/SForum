<?php

declare (strict_types=1);
namespace App\Command\CodeFec;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Psr\Container\ContainerInterface;

#[Command]
class MigratePackage extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
        parent::__construct('CodeFec:MigratePackage');
    }
    
    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec Migrate Package');
    }
    
    public function handle()
    {
        $this->plugins();
        $this->themes();
    }
    
    private function plugins()
    {
        $plugins = getPathDir(plugin_path());
        $rows = [];
        $time = time();
        foreach ($plugins as $key => $plugin) {
            $id = $key + 1;
            $plugin_name = $plugin;
            if (is_dir(plugin_path($plugin_name . "/resources/package"))) {
                if (!is_dir(BASE_PATH . "/resources/js/plugins")) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/js/plugins");
                }
                if (!is_dir(BASE_PATH . "/resources/sass/plugins")) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/sass/plugins");
                }
            }
            if (is_dir(plugin_path($plugin_name . "/resources/package/js"))) {
                if (!is_dir(BASE_PATH . "/resources/js/plugins/" . $plugin_name)) {
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/js/plugins/" . $plugin_name);
                }
                copy_dir(plugin_path($plugin_name . "/resources/package/js"), BASE_PATH . "/resources/js/plugins/" . $plugin_name);
            }
            if (is_dir(plugin_path($plugin_name . "/resources/package/sass"))) {
                if (!is_dir(BASE_PATH . "/resources/sass/plugins/" . $plugin_name)) {
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/sass/plugins/" . $plugin_name);
                }
                copy_dir(plugin_path($plugin_name . "/resources/package/sass"), BASE_PATH . "/resources/sass/plugins/" . $plugin_name);
            }
            $rows[] = [$id, $plugin, plugin_path($plugin_name), "<info>success</info>"];
        }
        $this->table(['插件id', '被执行插件', '插件路径', '迁移状态'], $rows, 'box-double');
    }
    
    private function themes()
    {
        $plugins = getPath(theme_path());
        $rows = [];
        $time = time();
        foreach ($plugins as $key => $plugin) {
            $id = $key + 1;
            $plugin_name = $plugin;
            if (is_dir(theme_path($plugin_name . "/resources/package"))) {
                if (!is_dir(BASE_PATH . "/resources/js/themes")) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/js/themes");
                }
                if (!is_dir(BASE_PATH . "/resources/sass/themes")) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/sass/themes");
                }
            }
            if (is_dir(theme_path($plugin_name . "/resources/package/js"))) {
                if (!is_dir(BASE_PATH . "/resources/js/themes/" . $plugin_name)) {
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/js/themes/" . $plugin_name);
                }
                copy_dir(theme_path($plugin_name . "/resources/package/js"), BASE_PATH . "/resources/js/themes/" . $plugin_name);
            }
            if (is_dir(theme_path($plugin_name . "/resources/package/sass"))) {
                if (!is_dir(BASE_PATH . "/resources/sass/themes/" . $plugin_name)) {
                    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/sass/themes/" . $plugin_name);
                }
                copy_dir(theme_path($plugin_name . "/resources/package/sass"), BASE_PATH . "/resources/sass/themes/" . $plugin_name);
            }
            $rows[] = [$id, $plugin, theme_path($plugin_name), "<info>success</info>"];
        }
        $this->table(['主题id', '被执行主题', '主题路径', '迁移状态'], $rows, 'box-double');
    }
}