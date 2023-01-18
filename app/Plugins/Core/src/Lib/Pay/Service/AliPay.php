<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib\Pay\Service;

use App\Plugins\Core\src\Controller\Pay\PayInterFace;
use App\Plugins\Core\src\Models\PayOrder;
use Yansongda\Pay\Pay;

class AliPay implements PayInterFace
{
    /**
     * 金额转元 -> 倍数.
     * @var float|int
     */
    private int | float $amount_multiple = 1;

    private array $pay_status = [
        'WAIT_BUYER_PAY' => '交易创建',
        'TRADE_FINISHED' => '交易完成',
        'TRADE_CLOSED' => '交易关闭',
        'TRADE_SUCCESS' => '支付成功',
    ];

    /**
     * 支付配置.
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
                    'return_url' => pay()->get_options('alipay_return_url', url('/pay/alipay/return')),
                    'notify_url' => pay()->get_options('alipay_notify_url', url('/api/pay/alipay/notify')),
                    // 选填-第三方应用授权token
                    //'app_auth_token' => '',
                    // 选填-服务商模式下的服务商 id，当 mode 为 Pay::MODE_SERVICE 时使用该参数
                    //'service_provider_id' => '',
                    // 选填-默认为正常模式。可选为： MODE_NORMAL, MODE_SANDBOX, MODE_SERVICE
                    'mode' => Pay::MODE_NORMAL,
                ],
            ],
            'logger' => [
                'enable' => env('PAY_LOG_ENABLE', true),
                'file' => BASE_PATH . '/runtime/logs/pay.log',
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
     * @throws \Psr\SimpleCache\InvalidArgumentException
     * @return \Yansongda\Pay\Provider\Alipay
     */
    public function pay()
    {
        return Pay::alipay(array_merge($this->config(), ['_force' => true]));
    }

    /**
     * 创建订单.
     * @throws \Psr\SimpleCache\InvalidArgumentException
     * @return array|\Yansongda\Supports\Collection
     */
    public function create(PayOrder $order): mixed
    {
        // 定义支付链接
        $url = url('/pay/alipay/' . $order->id . '/goto');
        // 如果是扫码付则重新定义
        if (get_options('alipay_pay_mode', 'MIX') === 'SCAN') {
            $create_order = [
                'out_trade_no' => (string) $order->id,
                'subject' => $order->title,
                'total_amount' => $this->calculate_amount($order->amount),
                'quit_url' => url(),
            ];
            $url = $this->pay()->scan($create_order)->qr_code;
        }
        return Json_Api(200, true, ['msg' => '订单创建成功!', 'url' => $url, 'order_id' => $order->id]);
    }

    /**
     * 支付回调.
     * @param $request
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\IdNotAllowedException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws \SleekDB\Exceptions\JsonException
     * @throws \Yansongda\Pay\Exception\ContainerException
     * @throws \Yansongda\Pay\Exception\InvalidParamsException
     * @return array|bool|\Psr\Http\Message\ResponseInterface
     */
    public function notify($request): bool | array | \Psr\Http\Message\ResponseInterface
    {
        $result = $this->pay()->callback($request)->toArray();
        //admin_log()->insert('Pay','WechatPay','回调结果',$result);
        $notify_result = pay()->notify(
            $result['out_trade_no'],
            $this->pay_status[$result['trade_status']],
            $result['trade_no'],
            $this->calculate_amount($result['receipt_amount'], true),
            $result,
            $this->calculate_amount($result['total_amount'], true),
            '[2,"aliPay"]'
        );
        if ($notify_result === true) {
            return true;
        }
        admin_log()->insert('Pay', 'AliPay', '支付回调失败!', $notify_result);
        return $notify_result;
    }

    /**
     * 查询订单.
     * @throws \Psr\SimpleCache\InvalidArgumentException
     * @throws \Yansongda\Pay\Exception\ContainerException
     * @throws \Yansongda\Pay\Exception\InvalidParamsException
     * @throws \Yansongda\Pay\Exception\ServiceNotFoundException
     */
    public function find(PayOrder $order): array
    {
        $result = $this->pay()->find([
            'out_trade_no' => $order->id,
            'trade_no' => $order->trade_no,
        ]);
        return [
            'amount' => $order->amount, //预收金额
            'payer_total' => $order->payer_total, //实收金额
            'status' => $this->pay_status[$result->trade_status], // 订单状态
            'amount_total' => $result->total_amount, // 订单总金额
            'success_time' => @$result->send_pay_date ?: null, // 支付完成时间
        ];
    }

    /**
     * 关闭订单.
     * @throws \Yansongda\Pay\Exception\ContainerException
     * @throws \Yansongda\Pay\Exception\InvalidParamsException
     * @throws \Yansongda\Pay\Exception\ServiceNotFoundException
     */
    public function close(PayOrder $order): array
    {
        $result = $this->pay()->close($order->id);
        if ((int) $result->code !== 10000) {
            admin_log()->insert('PayMent', 'AliPay', '关闭订单失败', $result);
            return Json_Api(500, false, ['msg' => '关闭订单失败,' . $result->sub_msg]);
        }
        PayOrder::query()->where('id', $order->id)->update([
            'status' => '交易关闭',
        ]);
        return Json_Api(200, true, ['msg' => '订单已关闭!']);
    }

    /**
     * 取消订单.
     * @throws \Yansongda\Pay\Exception\ContainerException
     * @throws \Yansongda\Pay\Exception\InvalidParamsException
     * @throws \Yansongda\Pay\Exception\ServiceNotFoundException
     */
    public function cancel(PayOrder $order): array
    {
        $result = $this->pay()->cancel($order->id);
        if ((int) $result->code !== 10000) {
            admin_log()->insert('PayMent', 'AliPay', '取消订单失败', $result);
            return Json_Api(500, false, ['msg' => '取消订单失败,' . $result->msg]);
        }
        PayOrder::query()->where('id', $order->id)->update([
            'status' => '订单取消',
        ]);
        return Json_Api(200, true, ['msg' => '取消订单成功!']);
    }

    /**
     * 计算实际金额.
     * @param int|string $amount
     * @param bool $dividing
     * @return float|int
     */
    protected function calculate_amount(string | int $amount, bool $dividing = false): float | int
    {
        if (! is_numeric($amount)) {
            return 0;
        }
        if ($dividing === true) {
            return $amount / $this->amount_multiple;
        }
        return $amount * $this->amount_multiple;
    }
}
