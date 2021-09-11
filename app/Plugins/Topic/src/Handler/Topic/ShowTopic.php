<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;

class ShowTopic
{
    public function handle($id): \Psr\Http\Message\ResponseInterface
    {
        // 自增浏览量
        go(static function() use ($id){
            Topic::query()->where('id', $id)->increment('view');
        });
        $data = Topic::query()
            ->where('id', $id)
            ->with("tag","user")
            ->first();
        return view('plugins.Core.topic.show.show',['data' => $data]);
    }
}