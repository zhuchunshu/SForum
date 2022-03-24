<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicLike;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;
use Hyperf\Utils\Arr;

#[Controller(prefix:"/api/topic")]
#[RateLimit(create:5, capacity:12)]
class ApiController
{
    #[PostMapping(path:"tags")]
    public function Tags(): array
    {
        $data = [];
        foreach (TopicTag::query()->get() as $key=>$value){
            $class_name = UserClass::query()->where('id',auth()->data()->class_id)->first()->name;
            if(user_TopicTagQuanxianCheck($value,$class_name)){
                $data = Arr::add($data,$key,[
                    "text"=>$value->name,
                    "value" => $value->id,
                    "icons" => "&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url($value->icon)&quot;&gt;&lt;/span&gt;"
                ]);
            }
        }
        return $data;
    }

    #[PostMapping(path:"keywords")]
    public function topic_keyword(): array
    {
        $data = TopicKeyword::query()->select('name','id')->get();
        $arr = [];
        foreach ($data as $key=>$value){
            $arr = Arr::add($arr,$key,["value"=>".[".$value->name."]","html" => $value->name]);
        }
        return $arr;
    }

    #[PostMapping(path:"with_topic.data")]
    public function get_WithTopic_Data(){
        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['请求的 帖子id不存在']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()) {
            return Json_Api(404,false,['ID为:'.$topic_id.'帖子不存在']);
        }

        $data = Topic::query()->where("id",$topic_id)->select("id","title","user_id","options","created_at")->with("user")->first();
        $user_avatar = super_avatar($data->user);
        $title = \Hyperf\Utils\Str::limit($data->title,20);
        $username = $data->user->username;
        $summary = \Hyperf\Utils\Str::limit(core_default(deOptions($data->options)["summary"],"未捕获到本文摘要"),40);
        return Json_Api(200,true,[
            "avatar" => $user_avatar,
            "title" => $title,
            "summary" => $summary,
            "username" => $username
        ]);
    }

    #[PostMapping("like.topic")]
    #[RateLimit(create:1, capacity:3)]
    public function like_topic(){

        if(!auth()->check()){
            return Json_Api(403,false,['msg' => '未登录!']);
        }

        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数:topic_id 不存在!']);
        }
        if(!Topic::query()->where([['id',$topic_id,'status'=>'publish']])->exists()) {
            return Json_Api(403,false,['msg' => 'id为:'.$topic_id."的帖子不存在"]);
        }
        if(TopicLike::query()->where(['topic_id'=>$topic_id,'user_id'=>auth()->id(),"type" => 'like'])->exists()) {
            TopicLike::query()->where(['topic_id'=>$topic_id,'user_id'=>auth()->id(),"type" => 'like'])->delete();
            Topic::query()->where(['id'=>$topic_id])->decrement("like");
	        // 发送通知
	        $topic_data = Topic::query()->where('id', $topic_id)->first();
	        if($topic_data->user_id!=auth()->id()){
		        $title = auth()->data()->username."对你的帖子取消了点赞";
		        $content = view("Topic::Notice.nolike_topic",['user_data' => auth()->data(),'data' => $topic_data]);
		        $action = "/".$topic_data->id.".html";
		        user_notice()->send($topic_data->user_id,$title,$content,$action);
	        }
            return Json_Api(201,true,['msg' => '已取消点赞']);
        }
        TopicLike::query()->create([
            "topic_id" => $topic_id,
            "user_id" => auth()->id(),
        ]);
        Topic::query()->where(['id'=>$topic_id])->increment("like");
	    // 发送通知
	    $topic_data = Topic::query()->where('id', $topic_id)->first();
	    if($topic_data->user_id!=auth()->id()){
		    $title = auth()->data()->username."赞了你的帖子";
		    $content = view("Topic::Notice.like_topic",['user_data' => auth()->data(),'data' => $topic_data]);
		    $action = "/".$topic_data->id.".html";
		    user_notice()->send($topic_data->user_id,$title,$content,$action);
	    }
        return Json_Api(200,true,['msg' =>'已赞!']);
    }

    #[PostMapping(path:"topic.data")]
    public function topic_data(){
        if(!auth()->check()){
            return Json_Api(403,false,['msg' => '未登录!']);
        }
        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:topic_id']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()){
            return Json_Api(403,false,['msg' => 'id为:'.$topic_id."的帖子不存在"]);
        }
        $data = Topic::query()->where("id",$topic_id)
            ->with("user","tag")
            ->first();
        $options = deOptions($data->options);
        $data['options'] = $options;
        return Json_Api(200,true,$data);
    }

    // 设置精华
    #[PostMapping(path:"set.topic.essence")]
    public function set_topic_essence(): array
    {
        $topic_id = request()->input("topic_id");
        $zhishu = request()->input("zhishu");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:topic_id']);
        }
        if(empty($zhishu) && $zhishu!=0){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:zhishu']);
        }
        if(!auth()->check() || !Authority()->check("topic_options")){
            return Json_Api(401,false,['msg' => '权限不足!']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()){
            return Json_Api(403,false,['msg' => '被操作的帖子不存在']);
        }
        if(!is_numeric($zhishu)){
            return Json_Api(403,false,['msg' => '精华指数必须为数字']);
        }
        if($zhishu<0 || $zhishu>999){
            return Json_Api(403,false,['msg' => '精华指数 必须大于或等于0 并且小于或等于999']);
        }
        if($zhishu===0){
            Topic::query()->where("id",$topic_id)->update([
                "essence" => null
            ]);
        }else{
            Topic::query()->where("id",$topic_id)->update([
                "essence" => $zhishu
            ]);
        }
        return Json_Api(200,true,['msg' => '更新成功!']);
    }

    // 设置置顶
    #[RateLimit(create:1, capacity:3)]
    #[PostMapping(path:"set.topic.topping")]
    public function set_topic_topping(): array
    {
        $topic_id = request()->input("topic_id");
        $zhishu = request()->input("zhishu");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:topic_id']);
        }
        if($zhishu!=0 && empty($zhishu)){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:zhishu']);
        }
        if(!auth()->check() || !Authority()->check("topic_options")){
            return Json_Api(401,false,['msg' => '权限不足!']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()){
            return Json_Api(403,false,['msg' => '被操作的帖子不存在']);
        }
        if(!is_numeric($zhishu)){
            return Json_Api(403,false,['msg' => '置顶指数必须为数字']);
        }
        if($zhishu<0 || $zhishu>999){
            return Json_Api(403,false,['msg' => '置顶指数 必须大于或等于0 并且小于或等于999']);
        }
        if($zhishu===0){
            Topic::query()->where("id",$topic_id)->update([
                "topping" => null
            ]);
        }else{
            Topic::query()->where("id",$topic_id)->update([
                "topping" => $zhishu
            ]);
        }
        return Json_Api(200,true,['msg' => '置顶成功!']);
    }

    // 删除帖子
    #[RateLimit(create:1, capacity:3)]
    #[PostMapping(path:"set.topic.delete")]
    public function set_topic_delete(): array
    {
        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:topic_id']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()){
            return Json_Api(403,false,['msg' => '被删除的帖子不存在']);
        }
        $data = Topic::query()->where("id",$topic_id)->first();
        $quanxian = false;
        if(Authority()->check("admin_topic_edit") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value']){
            $quanxian =true;
        }else if(Authority()->check("topic_edit") && auth()->id() === $data->user->id){
            $quanxian =true;
        }

        if(!auth()->check() || $quanxian===false){
            return Json_Api(401,false,['msg' => '权限不足!']);
        }
        Topic::query()->where("id",$topic_id)->update([
            "status" => "delete"
        ]);
        return Json_Api(200,true,['msg' => '已删除!']);
    }

    #[PostMapping(path:"star.topic")]
    #[RateLimit(create:1, capacity:3)]
    public function star_topic():array{
        if(!auth()->check()){
            return Json_Api(401,false,['msg' => '权限不足!']);
        }
        $topic_id = request()->input("topic_id");
        if(!$topic_id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:topic_id']);
        }
        if(!Topic::query()->where("id",$topic_id)->exists()){
            return Json_Api(403,false,['msg' => '要收藏的帖子不存在']);
        }
        if(UsersCollection::query()->where(['type' => 'topic','type_id' => $topic_id,'user_id' => auth()->id()])->exists()){
            UsersCollection::query()->where(['type' => 'topic','type_id' => $topic_id,'user_id' => auth()->id()])->delete();
            return Json_Api(200,true,['msg' => '取消收藏成功!']);
        }
        $topic = Topic::query()->where("id",$topic_id)->first();
        UsersCollection::query()->create([
            'user_id' => auth()->id(),
            'type' => 'topic',
            'type_id' => $topic_id,
            'action' => '/'.$topic_id.'.html',
            'title' => "<b style='color:red'>帖子</b> ".$topic->title,
            'content' => view("User::Collection.topic",['topic' => $topic])
        ]);
        return Json_Api(200,true,['msg'=>'已收藏']);
    }
}