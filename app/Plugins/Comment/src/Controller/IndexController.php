<?php

namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix:"/comment")]
class IndexController
{
    #[GetMapping(path:"topic/{id}.md")]
    public function show_topic_comment($id){
        if(!TopicComment::query()->where("id",$id)->exists()){
            return admin_abort("页面不存在",404);
        }
        $data = TopicComment::query()->select("markdown")->where("id",$id)->first()->markdown;
        return response()->raw($data);
    }
}