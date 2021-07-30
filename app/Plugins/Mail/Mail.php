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
    public function handle(){

        $this->setting();
        $this->composer();
        $this->helpers();
    }

    public function composer(){
        require_once __DIR__."/vendor/autoload.php";
    }

    public function setting(){
        require_once __DIR__."/setting.php";
    }

    public function helpers()
    {
        require_once __DIR__."/helpers.php";
    }
}