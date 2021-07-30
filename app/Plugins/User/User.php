<?php


namespace App\Plugins\User;

/**
 * Class User
 * @link https://github.com/zhuchunshu/sf-user
 * @author zhuchunshu
 * @package 用户管理插件
 * @name User
 * @version 1.0.0
 */
class User
{
    public function handle(){
        $this->boot();
        $this->helpers();
        $this->menu();
    }

    public function menu(){
        include __DIR__."/menu.php";
    }

    public function helpers(){
        include __DIR__."/helpers.php";
    }

    public function boot(){
        include __DIR__."/bootstrap.php";
    }
}