<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;
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
        $options = [
            "hidden" => [
                "type" => $hidden_type,
                "users" => $hidden_user_list,
                "user_class" => $hidden_user_class
            ]
        ];
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
            "_token" => auth()->id()."_".Str::random()
        ]);
    }

    public function validate($request):array|bool
    {
        if (cache()->has("topic_create_time_" . auth()->id())) {
            $time = cache()->get("topic_create_time_" . auth()->id())-time();
            return Json_Api(401,false,['发帖过于频繁,请 '.$time." 秒后再试"]);
        }
        return true;
    }
}