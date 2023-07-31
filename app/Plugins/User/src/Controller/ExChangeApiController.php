<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller;

use App\Plugins\Core\src\Models\PayAmountRecord;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Arr;
#[Middleware(LoginMiddleware::class)]
#[Controller(prefix: '/api/user/exchange')]
class ExChangeApiController
{
    // 余额转金币
    #[PostMapping('moneyTo_golds')]
    public function moneyTo_golds() : array
    {
        $data = request()->input('data');
        $captcha = request()->input('captcha');
        if (!$data || !$captcha) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (!captcha()->check($captcha)) {
            return Json_Api(403, false, ['msg' => '验证码错误']);
        }
        $data = de_stringify(request()->input('data'));
        if (!Arr::has($data, 'moneyTo_golds_num')) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        // 兑换的金币数量
        $moneyTo_golds_num = $data['moneyTo_golds_num'];
        if (!is_numeric($moneyTo_golds_num) || $moneyTo_golds_num <= 0) {
            return Json_Api(403, false, ['msg' => '请求参数格式有误']);
        }
        $user = User::query()->with('Options')->find(auth()->id());
        // 最多能兑换的金币数量
        $dc = \Hyperf\Utils\Str::after($user->Options->money, '.');
        $dc = \Hyperf\Utils\Str::length($dc);
        $max = intval($user->Options->money * get_options('wealth_how_many_money_to_golds', '1'));
        if ($moneyTo_golds_num > $max) {
            return Json_Api(403, false, ['msg' => '超出最大兑换限制']);
        }
        // 不能花的钱
        $_money = round($user->Options->money - intval($user->Options->money * get_options('wealth_how_many_money_to_golds', '1')) / get_options('wealth_how_many_money_to_golds', '1'), $dc);
        // 找钱
        $give_change = (string) ($max - $moneyTo_golds_num) / get_options('wealth_how_many_money_to_golds', '1') + $_money;
        // 扣费
        $deduction = $user->Options->money - $give_change;
        // 用户信息
        $options_id = auth()->data()->options_id;
        $option = UsersOption::query()->find($options_id);
        PayAmountRecord::query()->create(['original' => $option->money, 'cash' => round($option->money - $deduction, $dc), 'user_id' => auth()->id(), 'remark' => '兑换' . get_options('wealth_golds_name', '金币')]);
        UsersOption::query()->where('id', $options_id)->update(['money' => round($option->money - $deduction, $dc), 'golds' => intval($option->golds + $moneyTo_golds_num)]);
        return Json_Api(201, true, ['msg' => '兑换成功!']);
    }
    // 金币转积分
    #[PostMapping('goldsTo_credit')]
    public function goldsTo_credit() : array
    {
        $data = request()->input('data');
        $captcha = request()->input('captcha');
        if (!$data || !$captcha) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (!captcha()->check($captcha)) {
            return Json_Api(403, false, ['msg' => '验证码错误']);
        }
        $data = de_stringify(request()->input('data'));
        if (!Arr::has($data, 'goldsTo_credit_num')) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        // 兑换的金币数量
        $goldsTo_credit_num = $data['goldsTo_credit_num'];
        if (!is_numeric($goldsTo_credit_num) || $goldsTo_credit_num <= 0) {
            return Json_Api(403, false, ['msg' => '请求参数格式有误']);
        }
        $user = User::query()->with('Options')->find(auth()->id());
        // 兑换比例
        $proportion = get_options('wealth_how_many_golds_to_credit', 10);
        // 最多能兑换的金币数量
        $max = intval($user->Options->golds * $proportion);
        $dc = \Hyperf\Utils\Str::after($user->Options->golds, '.');
        $dc = \Hyperf\Utils\Str::length($dc);
        if ($goldsTo_credit_num > $max) {
            return Json_Api(403, false, ['msg' => '超出最大兑换限制']);
        }
        // 不能花的钱
        $_money = round($user->Options->golds - intval($user->Options->golds * $proportion) / $proportion, $dc);
        // 找钱
        $give_change = (string) ($max - $goldsTo_credit_num) / $proportion + $_money;
        // 扣费
        $deduction = $user->Options->golds - $give_change;
        // 用户信息
        $options_id = auth()->data()->options_id;
        $option = UsersOption::query()->find($options_id);
        PayAmountRecord::query()->create(['original' => '【' . get_options('wealth_golds_name', '金币') . '】' . $option->golds, 'cash' => '【' . get_options('wealth_golds_name', '金币') . '】' . round($option->golds - $deduction, $dc), 'user_id' => auth()->id(), 'remark' => '兑换' . get_options('wealth_credit_name', '积分')]);
        UsersOption::query()->where('id', $options_id)->update(['golds' => round($option->golds - $deduction, $dc), 'credits' => intval($option->credits + $goldsTo_credit_num)]);
        return Json_Api(201, true, ['msg' => '兑换成功!']);
    }
    // 余额转积分
    #[PostMapping('moneyTo_credit')]
    public function moneyTo_credit() : array
    {
        $data = request()->input('data');
        $captcha = request()->input('captcha');
        if (!$data || !$captcha) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (!captcha()->check($captcha)) {
            return Json_Api(403, false, ['msg' => '验证码错误']);
        }
        $data = de_stringify(request()->input('data'));
        if (!Arr::has($data, 'moneyTo_credit_num')) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        // 兑换的金币数量
        $moneyTo_credit_num = $data['moneyTo_credit_num'];
        if (!is_numeric($moneyTo_credit_num) || $moneyTo_credit_num <= 0) {
            return Json_Api(403, false, ['msg' => '请求参数格式有误']);
        }
        $user = User::query()->with('Options')->find(auth()->id());
        // 兑换比例
        $proportion = get_options('wealth_how_many_money_to_credit', get_options('wealth_how_many_money_to_golds', '1') * get_options('wealth_how_many_golds_to_credit', 10));
        // 最多能兑换的金币数量
        $max = intval($user->Options->money * $proportion);
        $dc = \Hyperf\Utils\Str::after($user->Options->money, '.');
        $dc = \Hyperf\Utils\Str::length($dc);
        if ($moneyTo_credit_num > $max) {
            return Json_Api(403, false, ['msg' => '超出最大兑换限制']);
        }
        // 不能花的钱
        $_money = round($user->Options->money - intval($user->Options->money * $proportion) / $proportion, $dc);
        // 找钱
        $give_change = (string) ($max - $moneyTo_credit_num) / $proportion + $_money;
        // 扣费
        $deduction = $user->Options->money - $give_change;
        // 用户信息
        $options_id = auth()->data()->options_id;
        $option = UsersOption::query()->find($options_id);
        PayAmountRecord::query()->create(['original' => $option->money, 'cash' => round($option->money - $deduction, $dc), 'user_id' => auth()->id(), 'remark' => '兑换' . get_options('wealth_credit_name', '积分')]);
        UsersOption::query()->where('id', $options_id)->update(['money' => round($option->money - $deduction, $dc), 'credits' => intval($option->credits + $moneyTo_credit_num)]);
        return Json_Api(201, true, ['msg' => '兑换成功!']);
    }
}