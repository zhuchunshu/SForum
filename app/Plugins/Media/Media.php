<?php

namespace App\Plugins\Media;

/**
 * Class Search
 * @name Media
 * @version 1.0.0
 * @author Inkedus
 * @package 媒体库插件
 * @link https://github.com/zhuchunshu/sf-media
 */
class Media
{
    public function handler(){
        $this->boot();
    }

    public function boot(): void
    {
        require_once __DIR__."/bootstrap.php";
    }
}