<?php

declare(strict_types=1);

namespace App\Command\CodeFec;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
#[Command]
class MigrateAllResources extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('CodeFec:AllRm');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec Resources Migrate All');
    }

    public function handle()
    {
        $plugins = getPath(plugin_path());
        $rows = [];
        $time = time();
        foreach($plugins as $key=>$plugin){
            $id = $key+1;
            $plugin_name = $plugin;
            if (is_dir(BASE_PATH."/resources/views/plugins/".$plugin_name)) {
                if (!is_dir(plugin_path($plugin_name."/resources"))) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    exec("mkdir " . plugin_path($plugin_name."/resources"));
                }
                if (!is_dir(plugin_path($plugin_name."/resources/views"))) {
                    //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                    exec("mkdir " . plugin_path($plugin_name."/resources/views"));
                }
                copy_dir(BASE_PATH . "/resources/views/plugins/" . $plugin_name,plugin_path($plugin_name . "/resources/views"));
            }
            if (is_dir(public_path("plugins/".$plugin_name))) {
                if (!is_dir(plugin_path($plugin_name."/resources"))) {
                    exec("mkdir " . plugin_path($plugin_name."/resources"));
                }
                if (!is_dir(plugin_path($plugin_name."/resources/assets"))) {
                    exec("mkdir " . plugin_path($plugin_name."/resources/assets"));
                }
                copy_dir(public_path("plugins/" . $plugin_name),plugin_path($plugin_name . "/resources/assets"));
            }
            $rows[]=[
              $id,
              $plugin,
                plugin_path($plugin_name),
                "<info>success</info>"
            ];
        }
        $this->table(['插件id','被执行插件','插件路径','迁移状态'],$rows,'box-double');
    }
}
