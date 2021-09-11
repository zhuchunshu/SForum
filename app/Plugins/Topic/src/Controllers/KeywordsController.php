<?php

namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class KeywordsController
{
    public function index(){

    }

    #[GetMapping(path:"/keywords/{name}.html")]
    public function data($name){
        if(!TopicKeyword::query()->where("name",$name)->exists()) {
            return admin_abort("标签:".$name."不存在",404);
        }
        $data = TopicKeyword::query()->where("name",$name)->first();
        $page = TopicKeywordsWith::query()->with("topic")->where("with_id",$data->id)->paginate(15);
        return view("plugins.Topic.keywords.data",['data' => $data,'page' => $page]);
    }
}