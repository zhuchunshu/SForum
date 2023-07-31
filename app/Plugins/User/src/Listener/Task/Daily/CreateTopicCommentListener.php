<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Listener\Task\Daily;

use App\Plugins\User\src\Event\Task\Daily\CreateTopicComment;
use App\Plugins\User\src\Models\UsersAward;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;
#[Listener]
class CreateTopicCommentListener implements ListenerInterface
{
    public function listen() : array
    {
        return [CreateTopicComment::class];
    }
    public function process(object $event): void
    {
        // 获取用户id
        $user_id = auth()->id();
        // 判断是否开启了评论奖励功能
        if (get_hook_credit_options('create_topic_comment_check', 'true') !== 'true') {
            return;
        }
        go(function () use($user_id) {
            // 判断是否开启了评论随机积分奖励功能
            if (get_hook_credit_options('credit_create_topic_comment_checkbox', 'true') === 'true') {
                // 开启了评论随机积分奖励
                // 获取随机积分奖励范围
                $credit_create_topic_comment_min = get_hook_credit_options('credit_create_topic_comment_min', 10);
                $credit_create_topic_comment_max = get_hook_credit_options('credit_create_topic_comment_max', 100);
                // 获取随机积分奖励
                $credit_create_topic_comment = mt_rand((int) $credit_create_topic_comment_min, (int) $credit_create_topic_comment_max);
                user_option($user_id)->addCredits($credit_create_topic_comment);
                // 添加积分变动记录
                create_amount_record($user_id, get_user_assets_credits($user_id), get_user_assets_credits($user_id) + $credit_create_topic_comment, 'credits', $credit_create_topic_comment, null, '评论随机奖励');
            }
            // 判断是否开启了评论固定积分奖励
            if (get_hook_credit_options('credit_create_topic_comment_fix_checkbox', 'false') === 'true') {
                // 开启了评论固定积分奖励
                // 获取固定积分奖励
                $credit_create_topic_comment = (int) get_hook_credit_options('credit_create_topic_comment_fix', 10);
                user_option($user_id)->addCredits($credit_create_topic_comment);
                // 添加积分变动记录
                create_amount_record($user_id, get_user_assets_credits($user_id), get_user_assets_credits($user_id) + $credit_create_topic_comment, 'credits', $credit_create_topic_comment, null, '评论固定奖励');
            }
            // 判断是否开启了评论随机金币奖励
            if (get_hook_credit_options('golds_create_topic_comment_checkbox', 'false') === 'true') {
                // 开启了评论随机金币奖励
                // 获取随机金币奖励范围
                $gold_create_topic_comment_min = get_hook_credit_options('golds_create_topic_comment_min', 10);
                $gold_create_topic_comment_max = get_hook_credit_options('golds_create_topic_comment_max', 100);
                // 获取随机金币奖励
                $gold_create_topic_comment = mt_rand((int) $gold_create_topic_comment_min, (int) $gold_create_topic_comment_max);
                user_option($user_id)->addGolds($gold_create_topic_comment);
                // 添加金币变动记录
                create_amount_record($user_id, get_user_assets_gold($user_id), get_user_assets_gold($user_id) + $gold_create_topic_comment, 'golds', $gold_create_topic_comment, null, '评论随机奖励');
            }
            // 判断是否开启了评论固定金币奖励
            if (get_hook_credit_options('golds_create_topic_comment_fix_checkbox', 'false') === 'true') {
                // 开启了评论固定金币奖励
                // 获取固定金币奖励
                $gold_create_topic_comment = (int) get_hook_credit_options('golds_create_topic_comment_fix', 10);
                user_option($user_id)->addGolds($gold_create_topic_comment);
                // 添加金币变动记录
                create_amount_record($user_id, get_user_assets_gold($user_id), get_user_assets_gold($user_id) + $gold_create_topic_comment, 'golds', $gold_create_topic_comment, null, '评论固定奖励');
            }
            // 写奖励记录
            UsersAward::create(['user_id' => $user_id, 'name' => 'create_topic_comment']);
        });
    }
}