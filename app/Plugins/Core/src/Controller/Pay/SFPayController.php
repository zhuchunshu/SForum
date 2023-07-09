<?php

namespace App\Plugins\Core\src\Controller\Pay;

use App\Plugins\Core\src\Lib\Pay\Service\SFPay;
use App\Plugins\Core\src\Models\PayAmountRecord;
use App\Plugins\Core\src\Models\PayOrder;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;
use HyperfExt\Hashing\Hash;

#[Controller(prefix:'/pay/SFPay')]
#[Middleware(LoginMiddleware::class)]
class SFPayController
{
    #[GetMapping(path:"{order_id}/paying")]
    public function paying($order_id)
    {
        // 判断订单是否存在
        if (!PayOrder::query()->where('id', $order_id)->where('user_id', auth()->id())->exists()) {
            return admin_abort('订单不存在', 403);
        }
        $order = PayOrder::find($order_id);
        if ($order->status !== '待支付' && $order->status !== '待付款' && $order->status !== '未支付' && $order->status !== '未付款') {
            return Json_Api(403, false, ['msg' => $order->status,]);
        }
        $money = User::query()->where('id', $order->user_id)->with('options')->first()->options->money;
        if ($money<$order->amount) {
            return admin_abort(get_options('wealth_money_name', '余额').'不足! 请充值', 403);
        }
        return view('App::Pay.Paying.SFPay', ['order' => $order,'money' => $money]);
    }

    #[PostMapping(path:"{order_id}/paying")]
    public function paying_submit($order_id)
    {
        // 判断订单是否存在
        if (!PayOrder::query()->where('id', $order_id)->where('user_id', auth()->id())->exists()) {
            return redirect()->url('/pay/SFPay/'.$order_id.'/paying')->with('danger', '订单不存在')->go();
        }
        $order = PayOrder::find($order_id);
        if ($order->status !== '待支付' && $order->status !== '待付款' && $order->status !== '未支付' && $order->status !== '未付款') {
            //return Json_Api(403, false, ['msg' => $order->status,]);
            return redirect()->url('/pay/SFPay/'.$order_id.'/paying')->with('danger', $order->status)->go();
        }
        $money = User::query()->where('id', $order->user_id)->with('options')->first()->options->money;
        if ($money<$order->amount) {
            return redirect()->url('/pay/SFPay/'.$order_id.'/paying')->with('danger', get_options('wealth_money_name', '余额').'不足! 请充值')->go();
        }
        // 验证密码
        $password = User::query()->where('id', $order->user_id)->with('options')->first()->password;

        if (!Hash::check(request()->input('password'), $password)) {
            return redirect()->url('/pay/SFPay/'.$order_id.'/paying')->with('danger', '密码错误!')->go();
        }
        // 扣款
        if (!PayAmountRecord::query()->where([
            'original' => $money,
            'cash' => $money-$order->amount,
            'user_id' => auth()->id(),
            'order_id' => $order->id,
            'remark' => '购买:【'.$order->title.'】'
        ])->exists()) {
            PayAmountRecord::query()->create([
                'original' => $money,
                'cash' => $money-$order->amount,
                'user_id' => auth()->id(),
                'order_id' => $order->id,
                'remark' => '购买:【'.$order->title.'】'
            ]);
            // 扣钱
            $options_id = User::query()->where('id', $order->user_id)->with('options')->first()->options_id;
            UsersOption::query()->where('id',$options_id)->update([
                'money' => (string)$money-$order->amount
            ]);
        }
        // 触发回调
        (new SFPay())->notify($order->id);
        return redirect()->url('/user/order/'.$order_id.'.order')->with('success', '支付成功!')->go();
    }
}
