<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 *
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
namespace App\CodeFec\View;

use App\CodeFec\Plugins;
use App\Model\AdminPlugin;
use Hyperf\Utils\ApplicationContext;
use Hyperf\View\Engine\EngineInterface;
use Hyperf\ViewEngine\Contract\FactoryInterface;

class HyperfViewEngine implements EngineInterface
{
    public function render($template, $data, $config): string
    {
        /** @var FactoryInterface $factory */
        $factory = ApplicationContext::getContainer()->get(FactoryInterface::class);
		// 插件
        $plugin_list = (new Plugins())->getEnPlugins();
        foreach ($plugin_list as $value) {
            $factory->addNamespace($value,plugin_path($value."/resources/views"));
        }
		// 替换
		foreach(Themes()->get() as $namespace=>$hints){
			$factory->replaceNamespace($namespace,$hints);
		}
		// 主题
	    $name = get_options("theme","CodeFec");
	    $factory->replaceNamespace("App",theme_path($name."/resources/views"));
	    $factory->replaceNamespace("Core",theme_path($name."/resources/views"));
        return $factory->make($template,$data)->render();
    }


}
