<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Crontab;

use App\Plugins\Core\src\Models\PayOrder;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicUpdated;
use App\Plugins\User\src\Models\UsersNotice;
use Hyperf\Crontab\Annotation\Crontab;
use Swoole\Coroutine\System;

#[Crontab(name: 'CleanDatabase', rule: '0 *\/12 * * *', callback: 'execute', enable: [CleanDatabase::class, 'isEnable'], memo: '数据库垃圾清理')]
class CleanDatabase
{
    public function execute()
    {
        // 清理已删除的文章
        $this->topic();
        // 清理已读通知
//        $this->notice();
        // 清理超过一天未付款、取消、关闭的订单
        $this->order();
        // 清理admin_logger日志
        $this->admin_logger();
        // 清理帖子更新记录
        $this->topic_updated();
    }

    public function isEnable(): bool
    {
        return true;
    }

    // 清理已删除的文章
    private function topic()
    {
        foreach (Post::where('topic_id')->get(['id', 'topic_id']) as $item) {
            go(function () use ($item) {
                if (! Topic::query()->where('id', $item->topic_id)->exists()) {
                    Post::query()->where('id', $item->id)->delete();
                }
            });
        }
        Topic::query()->where('status', 'delete')->delete();
    }

    // 清理已读通知
//    private function notice()
//    {
//        UsersNotice::query()->where('status', 'read')->delete();
//    }

    // 清理订单
    private function order()
    {
        $data = [];
        foreach (PayOrder::query()->where('status', '待支付')
            ->orWhere('status', '订单取消')
            ->orWhere('status', '交易关闭')->get() as $value) {
            if (time() - strtotime($value->created_at) > 86400) {
                $data[] = $value->id;
            }
        }
        foreach ($data as $id) {
            PayOrder::where('id', $id)->delete();
        }
        //return $data;
    }

    /**
     * 清理admin_logger过期日志.
     */
    private function admin_logger(): void
    {
        foreach (scandir(BASE_PATH . '/runtime/logs/admin_logger_database') as $name) {
            if (is_dir(BASE_PATH . '/runtime/logs/admin_logger_database/' . $name) && $name !== (string) date('YmW') && $name !== '.' && $name !== '..') {
                System::exec('rm -rf ' . BASE_PATH . '/runtime/logs/admin_logger_database/' . $name);
            }
        }
    }

    /**
     * 清理帖子更新记录.
     */
    private function topic_updated(): void
    {
        foreach (Topic::query()->get('id') as $topic) {
            if (TopicUpdated::query()->where('topic_id', $topic->id)->count() > 10) {
                $all = TopicUpdated::query()->where('topic_id', $topic->id)->skip(10)->take(100)->get();
                foreach ($all as $id) {
                    TopicUpdated::query()->where('id', $id->id)->delete();
                }
            }
        }
    }
}
