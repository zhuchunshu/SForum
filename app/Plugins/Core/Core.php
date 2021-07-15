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
    }

    public function setting(){
        require_once __DIR__."/setting.php";
    }
}