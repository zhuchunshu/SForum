<?php

namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Comment\src\Model\TopicCommentLike;
use App\Plugins\Comment\src\Request\Topic\UpdateComment;
use App\Plugins\Comment\src\Request\TopicCreate;
use App\Plugins\Comment\src\Request\TopicReply;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;
use Psr\EventDispatcher\EventDispatcherInterface;

#[Controller(prefix:"/api/comment")]
#[RateLimit(create:1, capacity:3)]
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
	
	    // 原内容
	    $yhtml = $request->input('content');
		
        // 过滤xss
        $content = xss()->clean($request->input('content'));

        // 解析艾特
        $content = $this->topic_create_at($content);

        $data = TopicComment::query()->create([
           'topic_id' => $request->input("topic_id"),
            'content' => $content,
            'markdown' => $request->input('markdown'),
            'user_id' => auth()->id()
        ]);
	    // 艾特被回复的人
	    $this->at_user($data,$yhtml);

        //发布成功
        // 发送通知
        $topic_data = Topic::query()->where('id', $request->input('topic_id'))->first();
        if($topic_data->user_id!=auth()->id()){
            $title = auth()->data()->username."评论了你发布的帖子!";
            $content = view("Comment::Notice.comment",['comment' => $content,'user_data' => auth()->data(),'data' => $data]);
            $action = "/".$topic_data->id.".html";
            user_notice()->send($topic_data->user_id,$title,$content,$action);
        }
        cache()->set("comment_create_time_" . auth()->id(), time()+get_options("comment_create_time", 60),get_options("comment_create_time", 60));

        return Json_Api(200,true,['msg'=>'发表成功!','url' => "/".$data->topic_id.".html/".$data->id."?page=".get_topic_comment_page($data->id)]);
    }

    // 回复评论
    #[PostMapping(path:"comment.topic.reply")]
    public function topic_reply_create(TopicReply $request): bool|array
    {
        // 鉴权
        if ($this->topic_create_validation()!==true){
            return $this->topic_create_validation();
        }
        // 处理

	    // 原内容
	    $yhtml = $request->input('content');
	    
        // 过滤xss
        $content = xss()->clean($request->input('content'));

        // 解析艾特
        $content = $this->topic_create_at($content);

        $comment_id = request()->input('comment_id');
        $topic_id = TopicComment::query()->where("id",$comment_id)->first()->topic_id;
        $parent_id = TopicComment::query()->where("id",$comment_id)->first()->user_id;
        $data = TopicComment::query()->create([
            'parent_url' => request()->input("parent_url"),
            'topic_id' => $topic_id,
            'parent_id' => $comment_id,
            'content' => $content,
            'markdown' => $request->input('markdown'),
            'user_id' => auth()->id()
        ]);
        $data = TopicComment::query()->where("id",$data->id)->first();
	
	    // 艾特被回复的人
	    $this->at_user($data,$yhtml);

        //发布成功
        // 发送通知 - 帖子作者
        $topic_data = Topic::query()->where('id', $topic_id)->first();
        if($topic_data->user_id!=auth()->id() && $topic_data->user_id!=$parent_id){
            $title = auth()->data()->username."评论了你发布的帖子!";
            $content = view("Comment::Notice.comment",['comment' => $content,'user_data' => auth()->data(),'data' => $data]);
            $action = "/".$topic_data->id.".html";
            user_notice()->send($topic_data->user_id,$title,$content,$action);
        }
        // 发送通知 - 被回复的人
        if($topic_data->user_id!=auth()->id() && $parent_id!=auth()->id()){
            $title = auth()->data()->username."回复了你的评论!";
            $content = view("Comment::Notice.reply",['comment' => $content,'user_data' => auth()->data(),'data' => $data]);
            $action = "/".$topic_data->id.".html";
            user_notice()->send($parent_id,$title,$content,$action);
        }
	 
		
		// 设置下一此回复时间
        cache()->set("comment_create_time_" . auth()->id(), time()+get_options("comment_create_time", 60),get_options("comment_create_time", 60));

        return Json_Api(200,true,['msg'=>'回复成功!','url' => "/".$data->topic_id.".html/".$data->id."?page=".get_topic_comment_page($data->id)]);
    }
	
	private function at_user(\Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder $data, string $html): void
	{
		$at_user = get_all_at($html);
		foreach($at_user as $username){
			go(function() use ($username,$data){
				if(User::query()->where("username",$username)->exists()){
					$user = User::query()->where("username",$username)->first();
					if((int)$user->id!==(int)$data->user_id){
						user_notice()->send(
							$user->id,
							"有人在评论中提到了你",
							"有人在评论中提到了你",
							"/".$data->topic_id.".html/".$data->id."?page=".get_topic_comment_page($data->id)
						);
					}
				}
			});
		}
	}

    // 删除帖子评论
    #[PostMapping(path:"comment.topic.delete")]
    public function topic_delete(){
        $comment_id = request()->input("comment_id");
        if(!auth()->check()){
            return Json_Api(401,false,['未登录']);
        }
        if(!$comment_id){
            return Json_Api(403,false,['请求参数不足,缺少:comment_id']);
        }
        if(!TopicComment::query()->where("id",$comment_id)->exists()){
            return Json_Api(403,false,['id为'.$comment_id."的评论不存在"]);
        }
        $data = TopicComment::query()->where("id",$comment_id)->with('user')->first();
        $quanxian = false;
        if(Authority()->check("admin_comment_remove") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value']){
            $quanxian = true;
        }
        if(Authority()->check("comment_remove") && auth()->id() === $data->user->id){
            $quanxian = true;
        }
        if($quanxian === false){
            return Json_Api(401,false,['无权限!']);
        }
        TopicComment::query()->where("id",$comment_id)->delete();
        return Json_Api(200,true,['已删除!']);
    }

    // 对帖子进行评论 -- 鉴权
    public function topic_create_validation(): bool|array
    {
        if(!auth()->check()){
            return Json_Api(401,false,['未登录']);
        }
        if(!Authority()->check("comment_create")){
            return Json_Api(401,false,['无评论权限']);
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

    #[PostMapping("like.topic.comment")]
    public function like_topic_comment(): array
    {
        if(!auth()->check()){
            return Json_Api(403,false,['msg' => '未登录!']);
        }

        $comment_id = request()->input("comment_id");
        if(!$comment_id){
            return Json_Api(403,false,['msg' => '请求参数:comment_id 不存在!']);
        }
        if(!TopicComment::query()->where('id',$comment_id)->exists()) {
            return Json_Api(403,false,['msg' => 'id为:'.$comment_id."的评论不存在"]);
        }
        if(TopicCommentLike::query()->where(['comment_id' => $comment_id,'user_id'=>auth()->id()])->exists()) {
            TopicCommentLike::query()->where(['comment_id' => $comment_id,'user_id'=>auth()->id()])->delete();
            TopicComment::query()->where(['id'=>$comment_id])->decrement("likes");
            return Json_Api(201,true,['msg' =>'已取消点赞!']);
        }
        TopicCommentLike::query()->create([
            "comment_id" => $comment_id,
            "user_id" => auth()->id(),
        ]);
        TopicComment::query()->where(['id'=>$comment_id])->increment("likes");
        return Json_Api(200,true,['msg' =>'已赞!']);
    }

    // 解析创建帖子评论的艾特内容
    private function topic_create_at(string $content)
    {
        return replace_all_at($content);
    }

    #[PostMapping(path:"topic.comment.data")]
    public function topic_comment_data(): array
    {
        $comment_id = request()->input('comment_id');
        if(!$comment_id){
            return Json_Api(403,false,['请求参数不足,缺少:comment_id']);
        }
        if(!TopicComment::query()->where([["id",$comment_id],['status','publish']])->exists()){
            return Json_Api(403,false,['id为:'.$comment_id."的评论不存在"]);
        }
        $data = TopicComment::query()
            ->where([["id",$comment_id],['status','publish']])
            ->first();
        return Json_Api(200,true,$data);
    }

    #[PostMapping(path:"topic.comment.update")]
    public function topic_comment_update(UpdateComment $request){
        $id = $request->input("comment_id"); // 获取评论id
        if(!TopicComment::query()->where("id",$id)->exists()){
            return Json_Api(404,false,["id为:".$id."的评论不存在"]);
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
            return Json_Api(401,false,["无权修改!"]);
        }
        // 过滤xss
        $content = xss()->clean($request->input('content'));

        // 解析艾特
        $content = $this->topic_create_at($content);
        $markdown = $request->input("markdown");
        TopicComment::query()->where(['id'=>$id])->update([
           "content" => $content,
            "markdown" => $markdown
        ]);
        return Json_Api(200,true,["更新成功!"]);
    }

    #[PostMapping("topic.caina.comment")]
    public function topic_caina_comment(){
        $comment_id = request()->input('comment_id');
        if(!$comment_id){
            return Json_Api(403,false,['请求参数不足,缺少:comment_id']);
        }
        if(!TopicComment::query()->where([["id",$comment_id],['status','publish']])->exists()){
            return Json_Api(403,false,['id为:'.$comment_id."的评论不存在"]);
        }
        $data = TopicComment::query()
            ->where([["id",$comment_id],['status','publish']])
            ->with("topic")
            ->first();
        $quanxian = false;
        if($data->topic->user_id == auth()->id() && Authority()->check("comment_caina")){
            $quanxian = true;
        }
        if(Authority()->check("admin_comment_caina")){
            $quanxian = true;
        }
        if($quanxian === false){
            return Json_Api(401,false,['无权限!']);
        }
        $caina = "取消采纳";
        if($data->optimal===null){
            TopicComment::query()->where([["id",$comment_id],['status','publish']])->update([
                "optimal" => date("Y-m-d H:i:s")
            ]);
            $caina = "采纳";
        }else{
            TopicComment::query()->where([["id",$comment_id],['status','publish']])->update([
                "optimal" => null
            ]);
        }
        if($data->user_id !== auth()->id()){
            $topic = Topic::query()->where("id",$data->topic_id)->first();
            user_notice()->send($data->user_id,auth()->data()->username.$caina."了你的评论","你发布在: <h2>".$topic->title."</h2> 的评论已被".$caina,"/".$topic->id.".html");
        }
        return Json_Api(200,true,['更新成功!']);
    }

	// 收藏评论
    #[PostMapping(path:"star.comment")]
    #[RateLimit(create:1, capacity:3)]
    public function star_topic():array{
        if(!auth()->check()){
            return Json_Api(401,false,['msg' => '权限不足!']);
        }
        $comment_id = request()->input("comment_id");
        if(!$comment_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:comment_id']);
        }
        if(!TopicComment::query()->where("id",$comment_id)->exists()){
            return Json_Api(403,false,['msg' => '要收藏的评论不存在']);
        }
        if(UsersCollection::query()->where(['type' => 'comment','type_id' => $comment_id,'user_id' => auth()->id()])->exists()){
            UsersCollection::query()->where(['type' => 'comment_id','type_id' => $comment_id,'user_id' => auth()->id()])->delete();
            return Json_Api(200,true,['msg' => '取消收藏成功!']);
        }
        UsersCollection::query()->create([
            'user_id' => auth()->id(),
            'type' => 'comment',
            'type_id' => $comment_id,
        ]);
        return Json_Api(200,true,['msg'=>'已收藏']);
    }
}