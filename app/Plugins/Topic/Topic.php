<?php


namespace App\Plugins\Topic;


/**
 * Class Topic
 * @name Topic
 * @author zhuchunshu
 * @link https://github.com/zhuchunshu/sf-topic
 * @package 帖子组件
 * @version 1.0.0
 */
class Topic
{
    public function handle(): void
    {

        $this->bootstrap();

    }

    public function bootstrap(): void
    {
        require_once __DIR__."/bootstrap.php";
    }
}