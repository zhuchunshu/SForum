<?php

namespace App\Plugins\Docs;

/**
 * @version 1.0.0
 * @package 文档
 * @name Docs
 * @author Inkedus
 * @link https://github.com/zhuchunshu/super-forum-docs
 */
class Docs
{
    public function handler(): void
    {
        $this->bootstrap();
    }

    public function bootstrap(): void
    {
        require_once __DIR__ . '/bootstrap.php';
    }
}