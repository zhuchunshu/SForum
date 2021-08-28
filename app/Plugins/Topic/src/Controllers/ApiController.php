<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\TopicTag;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Arr;

#[Controller(prefix:"/api/topic")]
class ApiController
{
    #[PostMapping(path:"tags")]
    public function Tags(): array
    {
        $data = [];
        foreach (TopicTag::query()->get() as $key=>$value){
            $data = Arr::add($data,$key,[
                "text"=>$value->name,
                "value" => $value->id,
                ]);
        }
        return $data;
    }
}