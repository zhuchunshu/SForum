<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Listener\Task\System;

use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersAward;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;

#[Listener]
class SetAvatar implements ListenerInterface
{
    public function listen(): array
    {
        return [
            \App\Plugins\User\src\Event\Task\System\SetAvatar::class,
        ];
    }

    public function process(object $event)
    {
        // 获取用户id
        $user_id = $event->user_id;
        // 判断是否开启上传头像奖励
        if (get_hook_credit_options('set_avatar_check', 'true') !== 'true') {
            return;
        }
        go(function () use ($user_id) {
            // 判断是否未给此用户发放奖励
            if (UsersAward::where('user_id', $user_id)->where('name', 'set_avatar')->exists()) {
                return;
            }
            // 判断是否开启积分奖励
            if (get_hook_credit_options('credit_set_avatar_checkbox', 'true') === 'true') {
                // 开启了上传头像积分奖励
                // 获取奖励积分
                $credit = get_hook_credit_options('credit_set_avatar', 100);
                // 奖励给用户积分
                $options_id = User::find($user_id)->options_id;
                $options = \App\Plugins\User\src\Models\UsersOption::find($options_id);
                $options->addCredits((int) $credit);
                create_amount_record($user_id, get_user_assets_credits($user_id), get_user_assets_credits($user_id) + $credit, 'credits', $credit, null, '上传头像奖励');
            }
            // 判断是否开启金币奖励
            if (get_hook_credit_options('golds_set_avatar_checkbox') === 'true') {
                // 开启了上传头像金币奖励
                // 获取奖励金币
                $gold = get_hook_credit_options('golds_set_avatar', 100);
                // 奖励给用户金币
                $options_id = User::find($user_id)->options_id;
                $options = \App\Plugins\User\src\Models\UsersOption::find($options_id);
                $options->addGolds((int) $gold);
                create_amount_record($user_id, get_user_assets_gold($user_id), get_user_assets_gold($user_id) + $gold, 'golds', $credit, null, '上传头像奖励');
            }

            // 写奖励记录
            UsersAward::create([
                'user_id' => $user_id,
                'name' => 'set_avatar',
            ]);
        });
    }
}
