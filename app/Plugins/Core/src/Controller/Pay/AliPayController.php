<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\Pay;

use App\Plugins\Core\src\Lib\Pay\Service\AliPay;
use App\Plugins\Core\src\Models\PayOrder;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use WhichBrowser\Parser;

#[Controller(prefix: '/pay/alipay')]
class AliPayController extends AliPay
{
    #[GetMapping(path: '{order_id}/goto')]
    public function create($order_id): mixed
    {
        $user_agent = request()->getHeader('user-agent')[0];
        if (! PayOrder::query()->where('id', $order_id)->exists()) {
            return Json_Api(403, false, ['msg' => '订单不存在']);
        }
        // 获取订单信息
        $order = PayOrder::query()->find($order_id);
        // 判断订单是否为未支付状态
        if ($order->status !== '待支付' && $order->status !== '待付款' && $order->status !== '未支付' && $order->status !== '未付款') {
            return Json_Api(403, false, ['msg' => $order->status]);
        }
        $create_order = [
            'out_trade_no' => (string) $order->id,
            'subject' => $order->title,
            'total_amount' => $this->calculate_amount($order->amount),
            'quit_url' => url(),
        ];

        // 选择支付方式
        switch (pay()->get_options('alipay_pay_mode', 'MIX')) {
            case 'WEB':
                $result = $this->pay()->web($create_order);
                break;
            case 'WAP':
                $result = $this->pay()->wap($create_order);
                break;
            default:
                if (! (new Parser($user_agent))->isMobile()) {
                    $result = $this->pay()->web($create_order);
                } else {
                    $result = $this->pay()->wap($create_order);
                }
                break;
        }

        return $result;
    }

    #[GetMapping(path: 'return')]
    public function return_notify()
    {
        return redirect()->url(url())->go();
    }
}
