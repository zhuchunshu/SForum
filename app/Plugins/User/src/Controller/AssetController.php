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
use App\Plugins\User\src\Middleware\AuthMiddleware;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Controller(prefix: '/user/asset')]
#[Middleware(LoginMiddleware::class)]
#[Middleware(AuthMiddleware::class)]
class AssetController
{
    #[GetMapping('record')]
    public function record()
    {
        $page = PayAmountRecord::query()->where('user_id', auth()->id())->whereNotNull('type')->where('type', '!=', 'money')->orderByDesc('created_at')->paginate(15);
        return view('User::assets.record', ['page' => $page]);
    }
    #[GetMapping('money')]
    public function money()
    {
        $page = PayAmountRecord::query()->where('type', null)->orWhere('type', 'money')->where('user_id', auth()->id())->orderByDesc('created_at')->paginate(15);
        return view('User::assets.money', ['page' => $page]);
    }
    #[PostMapping('money.recharge')]
    public function money_recharge_submit()
    {
        $payment = request()->input('payment');
        $amount = (float) request()->input('amount');
        $captcha = request()->input('captcha');
        if (!$payment || !$amount || !$captcha) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (!captcha()->check($captcha)) {
            return Json_Api(419, false, ['msg' => '验证码错误!']);
        }
        // 验证支付方式
        $payment = json_decode($payment, true);
        if (pay()->check_payment($payment) === true) {
            if ((int) $payment[0] === 0) {
                return Json_Api(403, false, ['msg' => '支付方式不可用']);
            }
        } else {
            return pay()->check_payment($payment);
        }
        // 支付方式可用，继续下一轮
        if (!is_numeric($amount) || $amount > 1000 || $amount < 0.01) {
            return Json_Api(403, false, ['msg' => '充值金额格式有误']);
        }
        // 发起支付
        $result = pay()->create(auth()->id(), auth()->data()->username . '账户充值', $amount, $payment);
        $order_id = $result['result']['order_id'];
        redis()->sAdd(env('APP_KEY', 'CodeFec') . ':' . 'user_pay_money_recharge', $order_id);
        return $result;
    }
}