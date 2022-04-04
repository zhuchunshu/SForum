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
        $this->plugins();
		$this->themes();
    }
	
	public function plugins(){
		$plugins = getPath(plugin_path());
		$rows = [];
		$time = time();
		foreach($plugins as $key=>$plugin){
			$id = $key+1;
			$plugin_name = $plugin;
			if (is_dir(public_path("plugins/".$plugin_name))) {
				if (!is_dir(plugin_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/assets"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources/assets"));
				}
				copy_dir(public_path("plugins/" . $plugin_name),plugin_path($plugin_name . "/resources/assets"));
			}
			
			if (is_dir(BASE_PATH."/resources/js/plugins/".$plugin_name)) {
				if (!is_dir(plugin_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/package"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources/package"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/package/js"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources/package/js"));
				}
				copy_dir(BASE_PATH."/resources/js/plugins/".$plugin_name,plugin_path($plugin_name . "/resources/package/js"));
			}
			
			if (is_dir(BASE_PATH."/resources/sass/plugins/".$plugin_name)) {
				if (!is_dir(plugin_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/package"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources/package"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/package/sass"))) {
					\Swoole\Coroutine\System::exec("mkdir " . plugin_path($plugin_name."/resources/package/sass"));
				}
				copy_dir(BASE_PATH."/resources/sass/plugins/".$plugin_name,plugin_path($plugin_name . "/resources/package/sass"));
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
	
	public function themes(){
		$plugins = getPath(theme_path());
		$rows = [];
		$time = time();
		foreach($plugins as $key=>$plugin){
			$id = $key+1;
			$plugin_name = $plugin;
			if (is_dir(theme_path("plugins/".$plugin_name))) {
				if (!is_dir(theme_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources"));
				}
				if (!is_dir(plugin_path($plugin_name."/resources/assets"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources/assets"));
				}
				copy_dir(public_path("themes/" . $plugin_name),theme_path($plugin_name . "/resources/assets"));
			}
			
			if (is_dir(BASE_PATH."/resources/js/themes/".$plugin_name)) {
				if (!is_dir(theme_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources"));
				}
				if (!is_dir(theme_path($plugin_name."/resources/package"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources/package"));
				}
				if (!is_dir(theme_path($plugin_name."/resources/package/js"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources/package/js"));
				}
				copy_dir(BASE_PATH."/resources/js/themes/".$plugin_name,theme_path($plugin_name . "/resources/package/js"));
			}
			
			if (is_dir(BASE_PATH."/resources/sass/themes/".$plugin_name)) {
				if (!is_dir(theme_path($plugin_name."/resources"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources"));
				}
				if (!is_dir(theme_path($plugin_name."/resources/package"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources/package"));
				}
				if (!is_dir(theme_path($plugin_name."/resources/package/sass"))) {
					\Swoole\Coroutine\System::exec("mkdir " . theme_path($plugin_name."/resources/package/sass"));
				}
				copy_dir(BASE_PATH."/resources/sass/themes/".$plugin_name,theme_path($plugin_name . "/resources/package/sass"));
			}
			$rows[]=[
				$id,
				$plugin,
				theme_path($plugin_name),
				"<info>success</info>"
			];
		}
		$this->table(['主题id','被执行主题','主题路径','迁移状态'],$rows,'box-double');
	}
}
