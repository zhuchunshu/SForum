<?php


namespace App\Plugins\Core;

/**
 * Class Core
 * @name Core
 * @author Inkedus
 * @link https://github.com/zhuchunshu/sf-core
 * @package 核心组件
 * @version 1.0.0
 */
class Core
{
    public function handle(){
        $this->setting();
        $this->helpers();
    }

    public function setting(){
        require_once __DIR__."/setting.php";
    }

    public function helpers(){
        require_once __DIR__."/helpers.php";
    }
}