<?php

namespace App\Plugins\Core\src\Lib\Pay\Service;

use App\Plugins\Core\src\Models\PayOrder;
use Yansongda\Pay\Pay;

class AliPay
{
    /**
     * 金额转分 -> 倍数
     * @var int|float
     */
    private int|float $amount_multiple = 1;

    /**
     * 计算实际金额
     * @param string|int $amount
     * @param bool $dividing
     * @return float|int
     */
    protected function calculate_amount(string|int $amount, bool $dividing=false): float|int
    {
        if(!is_numeric($amount)){
            return 0;
        }
        if($dividing===true){
            return $amount/$this->amount_multiple;
        }
        return $amount*$this->amount_multiple;
    }

    /**
     * 支付配置
     * @return array
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function config(): array
    {
        return [
            'alipay' => [
                'default' => [
                    // 必填-支付宝分配的 app_id
                    'app_id' => pay()->get_options('alipay_app_id'),
                    // 必填-应用私钥 字符串或路径
                    'app_secret_cert' => pay()->get_options('alipay_app_secret_cert'),
                    // 必填-应用公钥证书 路径
                    'app_public_cert_path' => pay()->get_options('alipay_app_public_cert_path'),
                    // 必填-支付宝公钥证书 路径
                    'alipay_public_cert_path' => pay()->get_options('alipay_public_cert_path'),
                    // 必填-支付宝根证书 路径
                    'alipay_root_cert_path' => pay()->get_options('alipay_root_cert_path'),
                    'return_url' => pay()->get_options('alipay_return_url',url('/pay/alipay/return')),
                    'notify_url' => pay()->get_options('alipay_notify_url',url('/api/pay/alipay/notify')),
                    // 选填-第三方应用授权token
                    //'app_auth_token' => '',
                    // 选填-服务商模式下的服务商 id，当 mode 为 Pay::MODE_SERVICE 时使用该参数
                    //'service_provider_id' => '',
                    // 选填-默认为正常模式。可选为： MODE_NORMAL, MODE_SANDBOX, MODE_SERVICE
                    'mode' => Pay::MODE_NORMAL,
                ]
            ],
            'logger' => [
                'enable' => env('PAY_LOG_ENABLE',false),
                'file' => BASE_PATH.'/runtime/logs/pay.log',
                'level' => 'debug', // 建议生产环境等级调整为 info，开发环境为 debug
                'type' => 'single', // optional, 可选 daily.
                'max_file' => 30, // optional, 当 type 为 daily 时有效，默认 30 天
            ],
            'http' => [ // optional
                'timeout' => 5.0,
                'connect_timeout' => 5.0,
                // 更多配置项请参考 [Guzzle](https://guzzle-cn.readthedocs.io/zh_CN/latest/request-options.html)
            ],
        ];

    }

    /**
     * 支付服务
     * @return \Yansongda\Pay\Provider\Alipay
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function pay()
    {
        return Pay::alipay(array_merge($this->config(), ['_force' => true]));
    }

    /**
     * 创建订单
     * @param PayOrder $order
     * @return array|\Yansongda\Supports\Collection
     */
    public function create($order){
        // 定义支付链接
        $url = url("/pay/alipay/".$order->id."/goto");
        // 如果是扫码付则重新定义
        if(get_options('alipay_pay_mode','MIX')==='SCAN'){
            $create_order = [
                'out_trade_no' => (string)$order->id,
                'subject' => $order->title,
                'total_amount' => $this->calculate_amount($order->amount),
                'quit_url' => url(),
            ];
            $url = $this->pay()->scan($create_order)->qr_code;
        }
        return Json_Api(200,true,['msg' => '订单创建成功!','url' => $url]);
    }

    private array $pay_status = [
        'WAIT_BUYER_PAY' => '交易创建',
        'TRADE_FINISHED' => '交易完成',
        'TRADE_CLOSED' => '交易关闭',
        'TRADE_SUCCESS' => '支付成功'
    ];

    /**
     * 支付回调
     * @param $request
     * @return array|bool|\Psr\Http\Message\ResponseInterface
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\IdNotAllowedException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws \SleekDB\Exceptions\JsonException
     * @throws \Yansongda\Pay\Exception\ContainerException
     * @throws \Yansongda\Pay\Exception\InvalidParamsException
     */
    public function notify($request): bool|array|\Psr\Http\Message\ResponseInterface
    {
        $result = $this->pay()->callback($request)->toArray();
        //admin_log()->insert('Pay','WechatPay','回调结果',$result);
        $notify_result = pay()->notify(
            $result['out_trade_no'],
            $this->pay_status[$result['trade_status']],
            $result['trade_no'],
            $this->calculate_amount($result['receipt_amount'],true),
            $result,
            $this->calculate_amount($result['total_amount'],true),
        );
        if($notify_result===true){
            return  true;
        }
        admin_log()->insert('Pay','AliPay','支付回调失败!',$notify_result);
        return $notify_result;
    }
}