<?php


namespace App\Plugins\Mail;

/**
 * Class Mail
 * @name Mail
 * @version 1.0.0
 * @package 发信邮箱组件
 * @author Inkedus
 * @link https://github.com/zhuchunshu/sf-mail
 */
class Mail
{
    public function handler(): void
    {

        $this->setting();
        $this->composer();
        $this->helpers();
    }

    public function composer(): void
    {
        require_once __DIR__."/vendor/autoload.php";
    }

    public function setting(): void
    {
        require_once __DIR__."/setting.php";
    }

    public function helpers(): void
    {
        require_once __DIR__."/helpers.php";
    }
}