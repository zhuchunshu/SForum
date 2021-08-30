<?php


namespace App\Plugins\Topic\src\Handler\Topic;


class CreateTopicView
{

    public function handler(): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Topic.create",['right' => $this->right()]);
    }

    // 右侧侧栏
    public function right(): array
    {
        Itf()->add("Topic_create_right",1,"plugins.Topic.create.right-quanxian");
        return Itf()->get("Topic_create_right");
    }
}