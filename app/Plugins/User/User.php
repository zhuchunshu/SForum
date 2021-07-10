<?php


namespace App\Plugins\User;

/**
 * Class User
 * @link https://github.com/zhuchunshu/sf-user
 * @author zhuchunshu
 * @description 用户管理插件
 * @name User
 * @version 1.0.0
 * @package App\Plugins\User
 */
class User
{
    public function handle(){
        $this->helpers();
        $this->menu();
    }

    public function menu(){
        include __DIR__."/menu.php";
    }

    public function helpers(){
        include __DIR__."/helpers.php";
    }
}