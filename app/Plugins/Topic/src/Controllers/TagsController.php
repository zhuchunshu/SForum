<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicTag;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class TagsController
{
    #[GetMapping(path: "/tags")]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->paginate(15,['*'],"TagPage");
        return view("plugins.Topic.Tags.index",['page' => $page]);
    }

    #[GetMapping(path: "/tags/{id}.html")]
    public function data($id){
        if(!TopicTag::query()->where('id',$id)->exists()) {
            return admin_abort("页面不存在",404);
        }
        $page = Topic::query()
            ->where("tag_id",$id)
            ->with("tag","user")
            ->orderBy("id","desc")
            ->paginate(get_options("topic_home_num",15));
        $data = TopicTag::query()->where("id",$id)->first();
        return view("plugins.Topic.Tags.data",['data' => $data,'page' => $page]);
    }
}