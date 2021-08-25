<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Handler\Topic\CreateTopicView;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix:"/topic")]
#[Middleware(\App\Plugins\User\src\Middleware\AuthMiddleware::class)]
class TopicController
{
    #[GetMapping(path: "create")]
    #[Middleware(LoginMiddleware::class)]
    public function create(){
        return (new CreateTopicView())->handler();
    }
}