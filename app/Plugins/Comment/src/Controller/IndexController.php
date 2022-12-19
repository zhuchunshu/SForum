<?php

namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Report;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix:"/comment")]
class IndexController
{
    #[GetMapping(path:"topic/{id}.md")]
    public function show_topic_comment($id){
	    if(get_options('comment_ban_markdown_preview')==="true"){
		    return admin_abort("页面不存在",404);
	    }
        if(Report::query()->where(['type' => 'comment','_id' => $id,'status' => 'approve'])->exists()){
            return admin_abort('此评论已被举报并批准,无法查看',403);
        }
        if(!TopicComment::query()->where("id",$id)->exists()){
            return admin_abort("页面不存在",404);
        }
        $data = TopicComment::query()->select("post_id")->where("id",$id)->first()->post->markdown;
	    return response()->raw(ShortCodeR()->filter($data));
    }

    #[GetMapping(path:"topic/{id}/edit")]
    public function edit_topic_comment($id){
        if(!TopicComment::query()->where("id",$id)->exists()){
            return admin_abort("id为:".$id."的评论不存在",404);
        }
        $data = TopicComment::query()->where("id",$id)->first();
        $quanxian = false;
        if(Authority()->check("admin_topic_edit") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value']){
            $quanxian = true;
        }
        if(Authority()->check("topic_edit") && auth()->id() === $data->user->id){
            $quanxian = true;
        }
        if($quanxian===false){
            return admin_abort("无权操作!",419);
        }
        return view("Comment::topic.edit",['data' => $data]);
    }
}