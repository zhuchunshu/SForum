<?php

namespace App\Plugins\Docs\src\Controllers;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix:"/admin/docs")]
#[Middleware(AdminMiddleware::class)]
class AdminController
{
    #[GetMapping(path:"")]
    public function index(){
        return view("Docs::admin.index");
    }
}