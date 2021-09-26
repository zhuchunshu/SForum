<?php

namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Comment\src\Request\TopicCreate;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix:"/api/comment")]
class ApiController
{
    // 对帖子进行评论
    #[PostMapping(path:"topic.create")]
    public function topic_create(TopicCreate $request){
        // 鉴权
        if ($this->topic_create_validation()!==true){
            return $this->topic_create_validation();
        }
        // 处理

        TopicComment::query()->create([
           'topic_id' => $request->input("topic_id"),
            'content' => $request->input('content'),
            'markdown' => $request->input('markdown'),
            'user_id' => auth()->id()
        ]);

        //发布成功
        cache()->set("comment_create_time_" . auth()->id(), time()+get_options("comment_create_time", 60),get_options("comment_create_time", 60));

        return Json_Api(200,true,['发表成功!']);
    }

    // 对帖子进行评论 -- 鉴权
    public function topic_create_validation(): bool|array
    {
        if(!auth()->check()){
            return Json_Api(401,false,['msg' => '未登录!']);
        }
        if(!Authority()->check("comment_create")){
            return Json_Api(401,false,['msg' => '无评论权限!']);
        }
        if (cache()->has("comment_create_time_" . auth()->id())) {
            $time = cache()->get("comment_create_time_" . auth()->id())-time();
            return Json_Api(401,false,['发表评论过于频繁,请 '.$time." 秒后再试"]);
        }
        return true;
    }

    #[PostMapping(path:"get.topic.comment")]
    public function topic_comment_list(){
        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['请求参数不足,缺少:topic_id']);
        }
        if(!Topic::query()->where(['status' => 'publish','id' => $topic_id])->exists()){
            return Json_Api(403,false,['ID为:'.$topic_id.'的帖子不存在']);
        }
        if(!TopicComment::query()->where(['status' => 'publish','topic_id'=>$topic_id])->count()){
            return Json_Api(403,false,['此帖子下无评论']);
        }
        $page = TopicComment::query()
            ->where(['status' => 'publish','topic_id'=>$topic_id])
            ->with('topic','user')
            ->paginate(get_options("comment_page_count",2));
        return Json_Api(200,true,$page);
    }
}