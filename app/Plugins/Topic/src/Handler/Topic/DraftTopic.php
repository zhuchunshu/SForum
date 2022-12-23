<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use Hyperf\Utils\Str;
use Psr\SimpleCache\InvalidArgumentException;

class DraftTopic
{
    public function handler($request){
        $this->create($request);
        $this->after();
        return Json_Api(200,true,['保存草稿成功!','2秒后跳转到网站首页']);
    }

    public function after(): void
    {

    }

    public function create($request): void
    {
        $title = $request->input("title");
        $tag = $request->input("tag");
        $html = $request->input("html");
        $summary = $request->input("summary");
        if(!$summary){
            $summary = remove_bbCode(strip_tags($html));
        }
        $images = getAllImg($html);
        $options = [
            "summary" => $summary,
            "images" => $images
        ];
        $html = xss()->clean($html);

        // 解析标签
        $yhtml = $html;
        $html = $this->tag($html);
        // 解析艾特
        $html = $this->at($html);

        $options = json_encode($options, JSON_THROW_ON_ERROR,JSON_UNESCAPED_UNICODE);
        $data = Topic::query()->create([
            "title" => $title,
            "user_id" => auth()->id(),
            "status" => "draft",
            "content" => $html,
            "like" => 0,
            "view" => 0,
            "tag_id" => $tag,
            "options" => $options,
            "_token" => auth()->id()."_".Str::random(),
            "updated_user" => auth()->id()
        ]);
        $this->topic_keywords($data,$yhtml);
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


}