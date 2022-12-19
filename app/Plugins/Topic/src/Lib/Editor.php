<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Lib;

class Editor
{
    /**
     * 获取编辑器插件列表.
     * @return bool|string
     */
    public static function plugins(): bool | string
    {
        $data = [];
        foreach (Itf()->get('topic-create-editor-plugins') as $value) {
            $data = array_merge($data, $value);
        }
        $data = array_unique($data);
        $data = array_values($data);
        return json_encode($data);
    }

    /**
     * 获取编辑器toolbar.
     * @return bool|string
     */
    public static function toolbar(): bool | string
    {
        $data = [];
        foreach (Itf()->get('topic-create-editor-toolbar') as $value) {
            $data = array_merge($data, $value);
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
        foreach (Itf()->get('topic-create-editor-menu') as $key => $value) {
            $data = array_merge($data, $value);
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
        foreach (Itf()->get('topic-create-editor-menu') as $key => $value) {
            $data = array_merge($data, $value);
        }
        $result = [];
        foreach ($data as $key => $value) {
            $result[] = $key;
        }
        $result = array_values(array_unique($result));
        return implode(' ',$result);
    }
}
