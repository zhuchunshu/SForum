<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec\View;

use App\CodeFec\Plugins;
use Hyperf\Di\Annotation\Inject;
use Hyperf\View\Engine\EngineInterface;
use Hyperf\ViewEngine\Contract\FactoryInterface;

class HyperfViewEngine implements EngineInterface
{
    #[Inject]
    protected FactoryInterface $factory;

    public function render($template, $data, $config): string
    {
        // 插件
        $plugin_list = (new Plugins())->getEnPlugins();
        foreach ($plugin_list as $value) {
            $this->factory->addNamespace($value, plugin_path($value . '/resources/views'));
        }
        // 主题
        $name = get_options('theme', 'CodeFec');
        $this->factory->replaceNamespace('App', theme_path($name . '/resources/views'));
        $this->factory->replaceNamespace('Core', theme_path($name . '/resources/views'));
        return $this->factory->make($template, $data)->render();
    }
}
