<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Api\Task;

use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\UsersAward;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Middleware(LoginMiddleware::class)]
#[Controller(prefix: '/api/user/task/checkin')]
class CheckinController
{
    #[PostMapping('')]
    public function checkin()
    {
        //判断是否开启了签到功能
        if (get_hook_credit_options('checkin_check', 'true') !== 'true') {
            return json_api(403, false, ['msg' => '签到功能未开启']);
        }
        // 判断用户今日是否已经签到
        if (\App\Plugins\User\src\Models\UsersAward::where('user_id', auth()->id())->where('name', 'checkin')->whereDate('created_at', \Carbon\Carbon::today())->exists()) {
            return json_api(403, false, ['msg' => '今日已签到']);
        }
        $award_credit = 0;
        $award_gold = 0;
        // 判断是否开启了签到随机积分奖励
        if (get_hook_credit_options('credit_checkin_checkbox', 'true') === 'true') {
            // 开启了签到随机积分奖励
            // 获取随机积分奖励范围
            $credit_checkin_min = get_hook_credit_options('credit_checkin_min', 10);
            $credit_checkin_max = get_hook_credit_options('credit_checkin_max', 100);
            // 获取随机积分奖励
            $credit_checkin = mt_rand((int) $credit_checkin_min, (int) $credit_checkin_max);
            user_option(auth()->id())->addCredits($credit_checkin);
            $award_credit += $credit_checkin;
            create_amount_record(auth()->id(), get_user_assets_credits(auth()->id()), get_user_assets_credits(auth()->id()) + $credit_checkin, 'credits', $credit_checkin, null, '签到随机奖励');
        }
        // 判断是否开启了签到固定积分奖励
        if (get_hook_credit_options('credit_checkin_fix_checkbox', 'false') === 'true') {
            // 开启了签到固定积分奖励
            // 获取固定积分奖励
            $credit_checkin_fix = (int) get_hook_credit_options('credit_checkin_fix', 10);
            user_option(auth()->id())->addCredits($credit_checkin_fix);
            $award_credit += $credit_checkin_fix;
            create_amount_record(auth()->id(), get_user_assets_credits(auth()->id()), get_user_assets_credits(auth()->id()) + $credit_checkin_fix, 'credits', $credit_checkin_fix, null, '签到固定奖励');
        }
        // 判断是否开启了签到随机金币奖励
        if (get_hook_credit_options('golds_checkin_checkbox', 'false') === 'true') {
            // 开启了签到金币奖励
            // 获取随机金币奖励范围
            $golds_checkin_min = get_hook_credit_options('golds_checkin_min', 10);
            $golds_checkin_max = get_hook_credit_options('golds_checkin_max', 100);
            // 获取随机金币奖励
            $golds_checkin = mt_rand((int) $golds_checkin_min, (int) $golds_checkin_max);
            user_option(auth()->id())->addGolds($golds_checkin);
            $award_gold += $golds_checkin;
            create_amount_record(auth()->id(), get_user_assets_gold(auth()->id()), get_user_assets_gold(auth()->id()) + $golds_checkin, 'golds', $golds_checkin, null, '签到随机奖励');
        }
        // 判断是否开启了签到固定金币奖励
        if (get_hook_credit_options('golds_checkin_fix_checkbox', 'false') === 'true') {
            // 开启了签到金币奖励
            // 获取固定金币奖励
            $golds_checkin_fix = (int) get_hook_credit_options('golds_checkin_fix', 10);
            user_option(auth()->id())->addGolds($golds_checkin_fix);
            $award_gold += $golds_checkin_fix;
            create_amount_record(auth()->id(), get_user_assets_gold(auth()->id()), get_user_assets_gold(auth()->id()) + $golds_checkin_fix, 'golds', $golds_checkin_fix, null, '签到固定奖励');
        }
        // 写奖励记录
        if (UsersAward::where('user_id', auth()->id())->where('name', 'checkin')->exists()) {
            UsersAward::where('user_id', auth()->id())->where('name', 'checkin')->update(['created_at' => \Carbon\Carbon::now()]);
        } else {
            UsersAward::create(['user_id' => auth()->id(), 'name' => 'checkin']);
        }
        $award = [];
        if ($award_gold > 0) {
            $award[get_options('wealth_golds_name', '金币')] = $award_gold;
        }
        if ($award_credit > 0) {
            $award[get_options('wealth_credits_name', '积分')] = $award_credit;
        }
        $_award = [];
        if (count($award)) {
            foreach ($award as $k => $item) {
                $_award[] = '获得 ' . $k . '奖励: ' . $item;
            }
        }
        if (count($_award)) {
            $award = "\n" . implode("\n", $_award);
            return json_api(200, true, ['msg' => $award]);
        }
        return json_api(200, true, ['msg' => '签到成功']);
    }
}