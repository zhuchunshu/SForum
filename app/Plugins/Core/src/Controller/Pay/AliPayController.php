<?php

namespace App\Plugins\Core\src\Controller\Pay;

use App\Plugins\Core\src\Lib\Pay\Service\AliPay;
use App\Plugins\Core\src\Models\PayOrder;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use WhichBrowser\Parser;

#[Controller(prefix: "/pay/alipay")]
class AliPayController extends AliPay
{
    #[GetMapping(path: "{order_id}/goto")]
    public function create($order_id)
    {
        $user_agent = request()->getHeader('user-agent')[0];
        $order = PayOrder::query()->find($order_id);
        $create_order = [
            'out_trade_no' => (string)$order->id,
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
                if (!(new Parser($user_agent))->isMobile()) {
                    $result = $this->pay()->web($create_order);
                } else {
                    $result = $this->pay()->wap($create_order);
                }
                break;
        }

        return $result;
    }

    #[GetMapping(path: "return")]
    public function return_notify()
    {
        return redirect()->url(url())->go();
    }
}