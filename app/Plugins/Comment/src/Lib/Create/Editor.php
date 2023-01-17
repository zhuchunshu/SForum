<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Lib\Create;

class Editor
{
    /**
     * 获取编辑器插件列表.
     * @return bool|string
     * @throws \JsonException
     */
    public static function plugins(): bool | string
    {
        $data = [];
        foreach (Itf()->get('comment-topic-create-editor-plugins') as $value) {
            foreach ($value as $plugin) {
                $data[] = $plugin;
            }
        }
        $data = array_unique($data);
        $data = array_values($data);
        return json_encode($data, JSON_THROW_ON_ERROR);
    }

    /**
     * 获取编辑器外部插件.
     * @return false|string
     */
    public static function externalPlugins(): bool | string
    {
        $data = [];
        foreach (Itf()->get('comment-topic-create-editor-external_plugins') as $value) {
            foreach ($value as $name => $plugin) {
                $data[$name] = $plugin;
            }
        }
        $data = array_unique($data);
        return json_encode($data);
    }

    /**
     * 获取编辑器toolbar.
     * @return bool|string
     */
    public static function toolbar(): bool | string
    {
        $data = [];
        foreach (Itf()->get('comment-topic-create-editor-toolbar') as $value) {
            foreach ($value as $toolbar) {
                $data[] = $toolbar;
            }
        }
        $data = array_values($data);
        $data = implode(' ', $data);
        return trim($data);
    }

    /**
     * 获取编辑器menu(菜单).
     */
    public static function menu()
    {
        $data = [];
        foreach (Itf()->get('comment-topic-create-editor-menu') as $key => $value) {
            foreach ($value as $keys => $menu) {
                $data[$keys] = $menu;
            }
        }
        $result = [];
        foreach ($data as $key => $value) {
            $value['items'] = implode(' ', $value['items']);
            $result[$key] = $value;
        }
        return json_encode($result);
    }

    /**
     * 获取编辑器menubar.
     */
    public static function menubar()
    {
        $data = [];
        foreach (Itf()->get('comment-topic-create-editor-menu') as $value) {
            foreach ($value as $key => $menu) {
                $data[$key] = $menu;
            }
        }
        foreach ($data as $key => $value) {
            $result[] = $key;
        }
        $result = array_values(array_unique($result));
        return implode(' ', $result);
    }
}
