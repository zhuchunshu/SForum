<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Topic\src\Models\Topic;

class ShowTopic
{
    public function handle($id): \Psr\Http\Message\ResponseInterface
    {
        $data = Topic::query()
            ->where('id', $id)
            ->with("tag","user")
            ->first();
        return view('plugins.Core.topic.show',['data' => $data]);
    }
}