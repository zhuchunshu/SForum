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
    public function handler(): void
    {
        $this->composer();
        $this->setting();
        $this->helpers();
        $this->shortCode();
    }

    public function setting(): void
    {
        require_once __DIR__."/setting.php";
    }

    public function helpers(): void
    {
        require_once __DIR__."/helpers.php";
    }

    public function composer(): void
    {
        require_once __DIR__."/vendor/autoload.php";
    }

    public function shortCode(): void{
        require_once __DIR__."/ShortCode.php";
    }
}