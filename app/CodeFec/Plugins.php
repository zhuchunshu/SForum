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

use App\Model\AdminPlugin;
use Noodlehaus\Config;

class Plugins
{
    public function GetAll(): array
    {
        $arr = getPath(plugin_path());
        $plugin_arr = [];
        foreach ($arr as $value) {
            if (file_exists(plugin_path($value . '/' . $value . '.php'))) {
                // 插件目录名
                $plugin_arr[$value]['dir'] = $value;
                // 插件路径
                $plugin_arr[$value]['path'] = plugin_path($value);
                // 插件类
                $plugin_arr[$value]['class'] = '\\App\\Plugins\\' . $value . '\\' . $value;
                // 插件信息
                $plugin_arr[$value]['data'] = [];
                if (file_exists(plugin_path($value . '/' . $value . '.json'))) {
                    $plugin_arr[$value]['data'] = array_merge($plugin_arr[$value]['data'], Config::load(plugin_path($value . '/' . $value . '.json'))->all());
                } else {
                    $plugin_arr[$value]['data'] = array_merge($plugin_arr[$value]['data'], get_plugins_doc($plugin_arr[$value]['class']));
                }

                $plugin_arr[$value]['file'] = plugin_path($value . '/' . $value . '.php');
            }
        }
        return $plugin_arr;
    }

    /**
     * @param string $dirName 插件目录名
     * @return ?string
     */
    public function has_logo(string $dirName): ?bool
    {
        // 插件logo
        $file = plugin_path($dirName . '/' . $dirName . '.png');
        if (file_exists($file)) {
            return true;
        }
        return false;
    }

    // 获取已启用的插件列表
    public function getEnPlugins()
    {
        return $this->get_all();
        $plugins = ['Core', 'Mail', 'User', 'Topic', 'Comment', 'Search'];
        if (! file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock')) {
            return $plugins;
        }
        if (! cache()->has('plugins.en')) {
            $array = AdminPlugin::query()->where('status', 1)->get();
            $result = [];
            foreach ($array as $value) {
                $result[] = $value->name;
            }
            $result = array_merge($plugins, $result);
            cache()->set('plugins.en', array_unique($result));
            return array_values(array_unique($result));
        }
        return array_values(array_unique(cache()->get('plugins.en')));
    }

    public function get_default()
    {
        return ['Core', 'Mail', 'User', 'Topic', 'Comment', 'Search'];
    }

    public function get_all()
    {
        $result = [];
        foreach ($this->GetAll() as $key => $plugin) {
            $result[] = $key;
        }
        $result = array_merge($result, $this->get_default());
        return array_values(array_unique($result));
    }

    public function get_all_data()
    {
        return $this->GetAll();
    }
}
