<?php
namespace App\Plugins\User\src\Controller;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

/**
 * Class UserClass
 * @Controller
 * @Middleware(\App\Middleware\AdminMiddleware::class)
 * @package App\Plugins\User\src\Controller
 */
class UserClass
{
    /**
     * @GetMapping(path="/admin/userClass")
     */
    public function index(){
        return view("plugins.User.Class.index");
    }

    /**
     * @GetMapping(path="/admin/userClass/create")
     */
    public function create(){
        return view("plugins.User.Class.create");
    }
}