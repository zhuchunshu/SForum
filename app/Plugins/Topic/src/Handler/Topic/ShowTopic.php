<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Utils\Str;

class ShowTopic
{
    public function handle($id)
    {
        // 自增浏览量
        go(static function() use ($id){
            $updated_at = Topic::query()->where('id', $id)->first()->updated_at;
            Topic::query()->where('id', $id)->increment('view',1,['updated_at' => $updated_at]);
        });
        $data = Topic::query()
            ->where('id', $id)
            ->with("tag","user","topic_updated","update_user")
            ->first();
        $this->session($data);
        return view('plugins.Core.topic.show.show',['data' => $data]);
    }

    public function session($data){
        if(!session()->has("view_topic_data")){
            session()->set("view_topic_data","view.topic.".Str::random());
        }
        if(!cache()->has(session()->get("view_topic_data"))){
            cache()->set(session()->get("view_topic_data"),$data,600);
        }
    }
}