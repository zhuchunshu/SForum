<?php

namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class TestController
{
    #[GetMapping(path:"/test")]
    public function test(){
        $content = Topic::query()->where("id",38)->first()->content;
        return get_all_at($content);
    }
}