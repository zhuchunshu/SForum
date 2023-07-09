<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Listener\Task\Daily;

use App\Plugins\User\src\Event\Task\Daily\CreateTopic;
use App\Plugins\User\src\Models\UsersAward;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;

#[Listener]
class CreateTopicListener implements ListenerInterface
{
    public function listen(): array
    {
        return [
            CreateTopic::class,
        ];
    }

    public function process(object $event)
    {
        $user = auth()->data();
        // 判断是否开启了发帖奖励功能
        if (get_hook_credit_options('create_topic_check', 'true') !== 'true') {
            return;
        }
        go(function () use ($user) {
            // 判断是否开启了发帖随机积分奖励
            if (get_hook_credit_options('credit_create_topic_checkbox', 'true') === 'true') {
                // 开启了发帖随机积分奖励
                // 获取随机积分奖励范围
                $credit_create_topic_min = get_hook_credit_options('credit_create_topic_min', 10);
                $credit_create_topic_max = get_hook_credit_options('credit_create_topic_max', 100);
                // 获取随机积分奖励
                $credit_create_topic = mt_rand((int) $credit_create_topic_min, (int) $credit_create_topic_max);
                user_option($user->id)->addCredits($credit_create_topic);
                // 添加积分变动记录
                create_amount_record($user->id, get_user_assets_credits($user->id), get_user_assets_credits($user->id) + $credit_create_topic, 'credits', $credit_create_topic, null, '发帖随机奖励');
            }
            // 判断是否开启了发帖固定积分奖励
            if (get_hook_credit_options('credit_create_topic_fix_checkbox', 'false') === 'true') {
                // 开启了发帖固定积分奖励
                // 获取固定积分奖励
                $credit_create_topic_fix = (int) get_hook_credit_options('credit_create_topic_fix', 10);
                user_option($user->id)->addCredits($credit_create_topic_fix);
                // 添加积分变动记录
                create_amount_record($user->id, get_user_assets_credits($user->id), get_user_assets_credits($user->id) + $credit_create_topic_fix, 'credits', $credit_create_topic_fix, null, '发帖固定奖励');
            }
            // 判断是否开启了发帖随机金币奖励
            if (get_hook_credit_options('golds_create_topic_checkbox', 'false') === 'true') {
                // 开启了发帖随机金币奖励
                // 获取随机金币奖励范围
                $golds_create_topic_min = get_hook_credit_options('golds_create_topic_min', 10);
                $golds_create_topic_max = get_hook_credit_options('golds_create_topic_max', 100);
                // 获取随机金币奖励
                $golds_create_topic = mt_rand((int) $golds_create_topic_min, (int) $golds_create_topic_max);
                user_option($user->id)->addGolds($golds_create_topic);
                // 添加金币变动记录
                create_amount_record($user->id, get_user_assets_gold($user->id), get_user_assets_gold($user->id) + $golds_create_topic, 'golds', $golds_create_topic, null, '发帖随机奖励');
            }
            // 盘带你是否开启了发帖固定金币奖励
            if (get_hook_credit_options('golds_create_topic_fix_checkbox', 'false') === 'true') {
                // 开启了发帖固定金币奖励
                // 获取固定金币奖励
                $golds_create_topic_fix = (int) get_hook_credit_options('golds_create_topic_fix', 10);
                user_option($user->id)->addGolds($golds_create_topic_fix);
                // 添加金币变动记录
                create_amount_record($user->id, get_user_assets_gold($user->id), get_user_assets_gold($user->id) + $golds_create_topic_fix, 'golds', $golds_create_topic_fix, null, '发帖固定奖励');
            }
            // 写奖励记录
            UsersAward::create([
                'user_id' => $user->id,
                'name' => 'create_topic',
            ]);
        });
    }
}
