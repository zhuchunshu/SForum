<?php

declare(strict_types=1);

namespace App\Command\CodeFec;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
class ResourcesMigrate extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('CodeFec:Rm');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec Resources Migrate');
    }

    public function handle()
    {
        $plugin_name = $this->ask("插件目录名");
        if (is_dir(BASE_PATH."/resources/views/plugins/".$plugin_name)) {
            if (!is_dir(plugin_path($plugin_name."/resources"))) {
                //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
                exec("mkdir " . plugin_path($plugin_name."/resources"));
            }
            // if (!is_dir(plugin_path($plugin_name."/resources/views"))) {
            //     //return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
            //     exec("mkdir " . plugin_path($plugin_name."/resources/views"));
            // }
            // copy_dir(BASE_PATH . "/resources/views/plugins/" . $plugin_name,plugin_path($plugin_name . "/resources/views"));
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
        $this->line($plugin_name."插件的资源复制成功!", 'info');
    }
}
