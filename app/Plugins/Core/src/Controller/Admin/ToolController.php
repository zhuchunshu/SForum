<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix: '/api/admin/tool')]
#[Middleware(AdminMiddleware::class)]
class ToolController
{
    /**
     * 获取冗余数据数量.
     */
    #[GetMapping(path: 'get_redundant_data_count')]
    public function get_redundant_data_count()
    {
        return count($this->get_redundant_data());
    }

    /**
     * 获取冗余数据数量.
     */
    #[PostMapping(path: 'clean_redundant_data')]
    public function clean_redundant_data(): array
    {
        go(function(){
            foreach (TopicComment::get() as $item) {
                if (! Topic::where('id', $item->topic_id)->exists()) {
                    // 清理数据
                    TopicComment::where('id', $item->id)->delete();
                }
            }
        });
        return json_api(200, true, ['msg' => '清理任务已创建']);
    }

    private function get_redundant_data(): array
    {
        $topic = [];
        foreach (TopicComment::get() as $item) {
            if (! Topic::where('id', $item->topic_id)->exists()) {
                $topic[] = $item->id;
            }
        }
        return $topic;
    }
}
