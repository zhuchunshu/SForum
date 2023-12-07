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
        $this->factory->replaceNamespace('App', theme_path('CodeFec/resources/views'));

        // 替换机制
        foreach ($this->getAllReplaceViews() as $namespace => $hints) {
            $this->factory->replaceNamespace($namespace, $hints);
        }

        return $this->factory->make($template, $data)->render();
    }

    private function getAllReplaceViews(): array
    {
        // 获取主题替换配置
        $trs = Itf()->get('theme-replace');
        // 按键名排序数组
        krsort($trs);
        // 初始化数组
        $arrays = [];
        // 遍历主题替换配置
        foreach ($trs as $item) {
            // 检查项是否为数组并包含指定键名
            if (is_array($item) && arr_has($item, ['namespace', 'path'])) {
                // 将路径添加到命名空间的数组中
                $arrays[$item['namespace']][] = $item['path'];
            }
        }
        // 检查是否存在 'App' 命名空间，若存在则添加额外的路径
        if (arr_has($arrays, 'App')) {
            $app = $arrays['App'];
            unset($arrays['App']);
            $app = array_merge($app, [theme_path('CodeFec/resources/views')]);
            $arrays = ['App' => $app] + $arrays;
        }
        // 获取所有插件
        $plugins = \plugins()->get_all();
        // 遍历插件数组
        foreach ($plugins as $pluginName) {
            // 检查数组中是否存在插件名对应的键
            if (arr_has($arrays, $pluginName)) {
                // 获取插件名对应的数组
                $app = $arrays[$pluginName];
                // 移除原数组中的插件名对应项
                unset($arrays[$pluginName]);
                // 合并插件路径到数组
                $app = array_merge($app, [plugin_path($pluginName . '/resources/views')]);
                // 将更新后的插件数组添加回原数组
                $arrays = [$pluginName => $app] + $arrays;
            }
        }
        // 返回最终数组
        return $arrays;
    }

}
