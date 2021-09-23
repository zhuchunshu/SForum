<?php

namespace App\Plugins\Topic\src\Handler\Topic;

class EditTopicView
{
    public function handler($data): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Topic.edit",['right' => $this->right(),'data' => $data]);
    }

    // 右侧侧栏
    public function right(): array
    {
        Itf()->add("Topic_create_right",1,"plugins.Topic.create.right-quanxian");
        Itf()->add("Topic_create_right",2,"plugins.Topic.create.right-summary");
        return Itf()->get("Topic_create_right");
    }
}