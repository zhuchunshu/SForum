<?php


namespace App\Plugins\Core\src\Controller;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

/**
 * @Controller
 * @package App\Plugins\Core\src\Controller
 */
class TestController
{
    /**
     * @GetMapping(path="/")
     */
    public function index(){
        return view("plugins.Core.test");
    }
}