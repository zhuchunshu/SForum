<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Handler\CreateTopicComment;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix: '/topic/create/comment')]
class CreateTopicCommentController
{
    #[PostMapping(path: '{topic_id}')]
    public function store($topic_id)
    {
        return (new CreateTopicComment())->handler($topic_id);
    }
}
