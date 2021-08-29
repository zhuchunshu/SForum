<?php


namespace App\Plugins\Topic\src\Handler\Topic;


class CreateTopicView
{
    public array $right = [
        "plugins.Topic.create.right-quanxian"
    ];

    public function handler(): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Topic.create",['right' => $this->right()]);
    }

    // 右侧侧栏
    public function right(): array
    {
        return $this->right;
    }
}