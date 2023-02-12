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

use App\Model\AdminOption;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Crontab\Annotation\Crontab;
use Hyperf\DbConnection\Db;
use Illuminate\Support\Carbon;

#[Crontab(name: 'SForumCrontab', rule: '0 *\/12 * * *', callback: 'execute', enable: [SForumCrontab::class, 'isEnable'], memo: 'SForum定时任务')]
class SForumCrontab
{
    public function execute()
    {
    }

    public function isEnable(): bool
    {
        return true;
    }

    // 自动锁定主题
    #[Crontab(rule: "0 *\/12 * * *", memo: "自动锁定主题")]
    public function topic_auto_lock()
    {
        // 已开启自动锁定主题
        if (AdminOption::where(['name' => 'topic_auto_lock', 'value' => 'true'])->exists()) {
            // 获取自动锁定主题的天数
            $topic_auto_lock_day = AdminOption::where(['name' => 'topic_auto_lock_day'])->value('value') ?: 30;
            foreach (Topic::where('created_at', '<', Carbon::now()->subDays($topic_auto_lock_day)->format('Y-m-d H:i:s'))->where('status', '!=', 'lock')->get() as $topic) {
                // 帖子发布天数
                // 锁帖
                Db::table('topic')->where('id', $topic->id)->update(['status' => 'lock']);

                // 发送通知

                $notice = new \App\Plugins\User\src\Lib\UserNotice();
                $notice->send($topic->user_id, '你有一条帖子已被系统定时任务自动锁定', '帖子《<a href="/' . $topic->id . '.html" >' . $topic->title . '</a>》已被系统定时任务发现并自动锁定，原因：帖子发布时间已大于系统设定自动锁帖时间', '/' . $topic->id . '.html', false,'system');
            }
        }
    }
}
