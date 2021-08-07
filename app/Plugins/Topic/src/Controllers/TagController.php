<?php


namespace App\Plugins\Topic\src\Controllers;


use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Middleware(AdminMiddleware::class)]
#[Controller]
class TagController
{
    #[GetMapping(path:"/admin/topic/tag/create")]
    public function create(){
        return view("plugins.Topic.Tag.create");
    }
}