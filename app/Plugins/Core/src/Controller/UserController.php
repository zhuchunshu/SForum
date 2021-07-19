<?php
namespace App\Plugins\Core\src\Controller;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

/**
 * Class UserController
 * @Controller
 * @package App\Plugins\Core\src\Controller
 */
class UserController
{
    /**
     * @GetMapping(path="/login")
     */
    public function login(){
        return plugins_core_captcha()->get();
    }

    /**
     * @GetMapping(path="/register")
     */
    public function register(){
        return view("plugins.Core.user.register");
    }
}