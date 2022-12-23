<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\CodeFec;

class CodeFec
{
    public function handle(): void
    {
        $this->plugins();
        $this->menu();
        $this->header();
        $this->boot();
        $this->themes();
        $this->setting();
        $this->route();
        $this->itf();
    }

    public function setting(): void
    {
        require BASE_PATH . '/app/CodeFec/Itf/Setting/default.php';
    }

    // 注册菜单
    public function menu(): void
    {
        require BASE_PATH . '/app/CodeFec/Menu/default.php';
    }

    //创建页头内容
    public function header(): void
    {
        require BASE_PATH . '/app/CodeFec/Header/default.php';
    }

    public function boot(): void
    {
        require BASE_PATH . '/app/CodeFec/bootstrap.php';
    }

    /**
     * 重写路由.
     */
    public function route(): void
    {
        require BASE_PATH . '/app/CodeFec/Itf/Route/default.php';
    }

    // 处理插件
    public function plugins(): void
    {
        $result = (new Plugins())->getEnPlugins();
        foreach ($result as $value) {
            if (file_exists(plugin_path($value . '/' . $value . '.php'))) {
                $class = '\\App\\Plugins\\' . $value . '\\' . $value;
                if (@method_exists(new $class(), 'handler')) {
                    (new $class())->handler();
                }
            }
        }
    }

    // 处理主题
    public function themes()
    {
        $name = get_options('theme', 'CodeFec'); //主题名
        if (file_exists(theme_path($name . '/' . $name . '.php'))) {
            $class = '\\App\\Themes\\' . $name . '\\' . $name;
            if (@method_exists(new $class(), 'handler')) {
                (new $class())->handler();
            }
        }
    }

    public function itf(): void
    {
        require BASE_PATH . '/app/CodeFec/Itf/Itf/default.php';
    }
}
