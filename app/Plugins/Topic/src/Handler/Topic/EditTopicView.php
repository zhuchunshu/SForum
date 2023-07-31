<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic;

class EditTopicView
{
    public function handler($data) : \Psr\Http\Message\ResponseInterface
    {
        return view('Topic::edit', ['right' => $this->right(), 'data' => $data]);
    }
    // 右侧侧栏
    public function right() : array
    {
        //Itf()->add("Topic_create_right",2,"Topic::create.right-summary");
        return Itf()->get('Topic_create_right');
    }
}