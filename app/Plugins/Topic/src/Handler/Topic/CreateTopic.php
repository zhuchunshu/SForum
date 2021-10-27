<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use Hyperf\Utils\Str;
use Psr\SimpleCache\InvalidArgumentException;

class CreateTopic
{
    public function handler($request){
        if($this->validate($request)!==true){
            return $this->validate($request);
        }
        $this->create($request);
        $this->after();
        return Json_Api(200,true,['发布成功!','2秒后跳转到网站首页']);
    }

    public function after(): void
    {
        try {
            cache()->set("topic_create_time_" . auth()->id(), time()+get_options("topic_create_time", 120),get_options("topic_create_time", 120));
        } catch (InvalidArgumentException $e) {
        }
    }

    public function create($request): void
    {
        $title = $request->input("title");
        $tag = $request->input("tag");
        $markdown = $request->input("markdown");
        $html = $request->input("html");
        $hidden_type = $request->input("options_hidden_type");
        $hidden_user_class = $request->input("options_hidden_user_class");
        $hidden_user_list = $request->input("options_hidden_user_list");
        $summary = $request->input("summary");
        if(!$summary){
            $summary = remove_bbCode(strip_tags($html));
        }
        if($hidden_user_class){
            $hidden_user_class = de_stringify($hidden_user_class);
        }else{
            $hidden_user_class = [];
        }

        if($hidden_user_list){
            $hidden_user_list = de_stringify($hidden_user_list);
        }else{
            $hidden_user_list = [];
        }
        if(!$hidden_type){
            $hidden_type = "close";
        }
        $images = getAllImg($html);
        $options = [
            "hidden" => [
                "type" => $hidden_type,
                "users" => $hidden_user_list,
                "user_class" => $hidden_user_class
            ],
            "summary" => $summary,
            "images" => $images
        ];
        $html = xss()->clean($html);
        // 解析shortCode
        $html = ShortCode()->handle($html);

        // 解析标签
        $yhtml = $html;
        $html = $this->tag($html);
        // 解析艾特
        $html = $this->at($html);

        $options = json_encode($options, JSON_THROW_ON_ERROR,JSON_UNESCAPED_UNICODE);
        $data = Topic::query()->create([
            "title" => $title,
            "user_id" => auth()->id(),
            "status" => "publish",
            "content" => $html,
            "markdown" => $markdown,
            "like" => 0,
            "view" => 0,
            "tag_id" => $tag,
            "options" => $options,
            "_token" => auth()->id()."_".Str::random(),
            "updated_user" => auth()->id()
        ]);
        $this->topic_keywords($data,$yhtml);
        $this->at_user($data,$yhtml);
    }

    public function validate($request):array|bool
    {
        if (cache()->has("topic_create_time_" . auth()->id())) {
            $time = cache()->get("topic_create_time_" . auth()->id())-time();
            return Json_Api(401,false,['发帖过于频繁,请 '.$time." 秒后再试"]);
        }
        $class_name = UserClass::query()->where('id',auth()->data()->class_id)->first()->name;
        $tag_value = TopicTag::query()->where("id",$request->input('tag'))->first();
        if(!user_TopicTagQuanxianCheck($tag_value,$class_name)){
            return Json_Api(401,false,['无权使用此标签']);
        }
        return true;
    }

    public function tag(string $html)
    {
        $html = replace_all_keywords($html);
        return $html;
    }

    public function at(string $html): string
    {
        return replace_all_at($html);
    }

    public function topic_keywords($data,string $html): void
    {
        foreach (get_all_keywords($html) as $tag){
            if(!TopicKeyword::query()->where("name",$tag)->exists()) {
                TopicKeyword::query()->create([
                    "name" => $tag,
                    "user_id" => auth()->id()
                ]);
            }
            $tk = TopicKeyword::query()->where("name",$tag)->first();
            if(!TopicKeywordsWith::query()->where(['topic_id'=>$data->id,'with_id' => $tk->id])->exists()) {
                TopicKeywordsWith::query()->create([
                    'topic_id' => $data->id,
                    'with_id' => $tk->id,
                    'user_id' => auth()->id()
                ]);
            }
        }
    }

    private function at_user(\Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder $data, string $html): void
    {
        $at_user = get_all_at($html);
        foreach($at_user as $value){
            go(function() use ($value,$data){
                if(User::query()->where("username",$value)->exists()){
                    $user = User::query()->where("username",$value)->first();
                    if($user->id!==$data->user_id){
                        user_notice()->send($user->id,"有人在帖子中提到了你",$user->username."在帖子<b>".$data->title."</b>中提到了你","/".$data->id.".html");
                    }
                }
            });
        }
    }


}