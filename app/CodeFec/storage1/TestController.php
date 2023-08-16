<?php

namespace App\CodeFec\storage;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix: "/test")]
class TestController
{
    #[GetMapping("")]
    public function index(){
        return [123];
    }
}