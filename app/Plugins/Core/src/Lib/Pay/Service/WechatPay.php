<?php

namespace App\Plugins\Core\src\Lib\Pay\Service;

use App\Plugins\Core\src\Controller\Pay\PayInterFace;
use App\Plugins\Core\src\Models\PayOrder;
use Psr\SimpleCache\InvalidArgumentException;
use Yansongda\Pay\Exception\{ContainerException, InvalidParamsException, ServiceNotFoundException};
use Yansongda\Pay\Pay;
class WechatPay implements PayInterFace
{
    /**
     * 金额转元 -> 倍数
     * @var int | float
     */
    private int|float $amount_multiple = 100;
    /**
     * 计算实际金额
     * @param string|int $amount
     * @param bool $dividing
     * @return float|int
     */
    protected function calculate_amount(string|int $amount, bool $dividing = false) : float|int
    {
        if (!is_numeric($amount)) {
            return 0;
        }
        if ($dividing === true) {
            return $amount / $this->amount_multiple;
        }
        return $amount * $this->amount_multiple;
    }
    /**
     * 支付配置
     * @return array
     * @throws InvalidArgumentException
     */
    public function config() : array
    {
        return ['wechat' => ['default' => [
            // 必填-商户号，服务商模式下为服务商商户号
            'mch_id' => pay()->get_options('wechat_mch_id'),
            // 必填-商户秘钥
            'mch_secret_key' => pay()->get_options('wechat_mch_secret_key'),
            // 必填-商户私钥 字符串或路径
            'mch_secret_cert' => pay()->get_options('wechat_mch_secret_cert'),
            // 必填-商户公钥证书路径
            'mch_public_cert_path' => pay()->get_options('wechat_mch_public_cert_path'),
            // 必填
            'notify_url' => pay()->get_options('wechat_notify_url', url('/api/pay/wechat/notify')),
            // 必填 微信公众号id
            'mp_app_id' => pay()->get_options('wechat_mp_app_id'),
            // 选填-默认为正常模式。可选为： MODE_NORMAL, MODE_SERVICE
            'mode' => Pay::MODE_NORMAL,
        ]], 'logger' => [
            'enable' => env('PAY_LOG_ENABLE', false),
            'file' => BASE_PATH . '/runtime/logs/pay.log',
            'level' => 'debug',
            // 建议生产环境等级调整为 info，开发环境为 debug
            'type' => 'single',
            // optional, 可选 daily.
            'max_file' => 30,
        ], 'http' => [
            // optional
            'timeout' => 5.0,
            'connect_timeout' => 5.0,
        ]];
    }
    /**
     * 支付服务
     * @return \Yansongda\Pay\Provider\Wechat
     * @throws InvalidArgumentException
     */
    public function pay() : \Yansongda\Pay\Provider\Wechat
    {
        return Pay::wechat(array_merge($this->config(), ['_force' => true]));
    }
    /**
     * 创建订单
     * @param PayOrder $order
     * @return array|\Yansongda\Supports\Collection
     * @throws InvalidArgumentException
     */
    public function create(PayOrder $order) : \Yansongda\Supports\Collection|array
    {
        $create_order = ['out_trade_no' => (string) $order->id, 'description' => $order->title, 'amount' => ['total' => $this->calculate_amount($order->amount)]];
        $result = $this->pay()->scan($create_order);
        return Json_Api(200, true, ['msg' => '订单创建成功!', 'url' => $result->code_url, 'order_id' => $order->id]);
    }
    /**
     * 支付回调
     * @param $request
     * @return array|bool|\Psr\Http\Message\ResponseInterface
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\IdNotAllowedException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws \SleekDB\Exceptions\JsonException
     * @throws ContainerException
     * @throws InvalidParamsException
     */
    public function notify($request) : bool|array|\Psr\Http\Message\ResponseInterface
    {
        $result = $this->pay()->callback($request)->toArray();
        //admin_log()->insert('Pay','WechatPay','回调结果',$result);
        $notify_result = pay()->notify($result['resource']['ciphertext']['out_trade_no'], $result['resource']['ciphertext']['trade_state_desc'], $result['resource']['ciphertext']['transaction_id'], $this->calculate_amount($result['resource']['ciphertext']['amount']['payer_total'], true), $result['resource']['ciphertext'], $this->calculate_amount($result['resource']['ciphertext']['amount']['total'], true), '[1,"wechatPay"]');
        if ($notify_result === true) {
            return true;
        }
        admin_log()->insert('Pay', 'WechatPay', '支付回调失败!', $notify_result);
        return $notify_result;
    }
    /**
     * 查询订单
     * @param PayOrder $order
     * @return array
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws ServiceNotFoundException|InvalidArgumentException
     */
    public function find(PayOrder $order) : array
    {
        $result = $this->pay()->find($order->trade_no);
        return [
            'amount' => $order->amount,
            //预收金额
            'payer_total' => $this->calculate_amount($result->amount['payer_total'], true),
            //实收金额
            'status' => $result->trade_state_desc,
            // 订单状态
            'amount_total' => $this->calculate_amount($result->amount['total'], true),
            // 订单总金额
            'success_time' => @$result->success_time ?: null,
        ];
    }
    /**
     * 关闭订单
     * @param PayOrder $order
     * @return array
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws ServiceNotFoundException
     */
    public function close(PayOrder $order) : array
    {
        $this->pay()->close($order->id);
        PayOrder::query()->where('id', $order->id)->update(['status' => '交易关闭']);
        return Json_Api(200, true, ['msg' => '订单已关闭!']);
    }
    /**
     * 取消订单
     * @param PayOrder $order
     * @return array
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws ServiceNotFoundException
     */
    public function cancel(PayOrder $order) : array
    {
        $this->pay()->close($order->id);
        PayOrder::query()->where('id', $order->id)->update(['status' => '订单取消']);
        return Json_Api(200, true, ['msg' => '取消订单成功!']);
    }
}