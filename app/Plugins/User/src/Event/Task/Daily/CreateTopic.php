<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Event\Task\Daily;

class CreateTopic
{
    public string|int $topic_id;
    public function __construct($topic_id)
    {
        $this->topic_id = $topic_id;
    }
}