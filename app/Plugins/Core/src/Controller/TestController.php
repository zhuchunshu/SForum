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

    #[GetMapping(path: "/test")]
    public function test()
    {

        return session()->get("auth_data_class");
    }
}