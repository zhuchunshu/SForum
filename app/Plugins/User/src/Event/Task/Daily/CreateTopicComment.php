<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Event\Task\Daily;

class CreateTopicComment
{
    public string | int $comment_id;

    public function __construct(string | int $comment_id)
    {
        $this->comment_id = $comment_id;
    }
}
