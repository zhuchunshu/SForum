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
        return view("plugins.Core.user.sign",['title' => "登陆","view" => "plugins.Core.user.login"]);
    }

    /**
     * @GetMapping(path="/register")
     */
    public function register(){
        return view("plugins.Core.user.sign",['title' => "注册","view" => "plugins.Core.user.register"]);
    }
}