<?php

declare (strict_types=1);
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
use App\Plugins\User\src\Models\User;
use Hyperf\Stringable\Str;
use Psr\Http\Message\ResponseInterface;
use Psr\SimpleCache\InvalidArgumentException;
use SleekDB\Exceptions\IdNotAllowedException;
use SleekDB\Exceptions\InvalidConfigurationException;
use SleekDB\Exceptions\IOException;
use SleekDB\Exceptions\JsonException;
use Yansongda\Pay\Exception\ContainerException;
use Yansongda\Pay\Exception\InvalidParamsException;
use Yansongda\Pay\Exception\ServiceNotFoundException;
class SFPay implements PayInterFace
{
    /**
     * 金额转元 -> 倍数.
     * @var int | float
     */
    private int|float $amount_multiple = 1;
    /**
     * 支付服务
     */
    public function pay()
    {
    }
    /**
     * 创建订单.
     * @param PayOrder $order
     * @throws InvalidArgumentException
     * @return array|float|int|\Yansongda\Supports\Collection
     */
    public function create(PayOrder $order) : mixed
    {
        // 账户余额
        $money = User::query()->where('id', $order->user_id)->with('options')->first()->options->money;
        if ($money < $order->amount) {
            return Json_Api(403, false, ['msg' => get_options('wealth_money_name', '余额') . '不足! 请充值', 'url' => '']);
        }
        return Json_Api(200, true, ['msg' => '订单创建成功!', 'url' => url('/pay/SFPay/' . $order->id . '/paying'), 'order_id' => $order->id]);
    }

    /**
     * 支付回调.
     * @param  $order_id
     * @return array|bool|ResponseInterface
     * @throws IOException
     * @throws IdNotAllowedException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws InvalidConfigurationException
     * @throws JsonException
     */
    public function notify($order_id) : bool|array|\Psr\Http\Message\ResponseInterface
    {
        $order = PayOrder::find($order_id);
        $notify_result = pay()->notify($order->id, '支付成功', Str::random(5) . date('Ymd') . auth()->id() . date('His'), $order->amount, [], $order->amount, '[0,"SFPay"]');
        if ($notify_result === true) {
            return true;
        }
        admin_log()->insert('Pay', 'SFPay', '支付回调失败!', $notify_result);
        return $notify_result;
    }
    /**
     * 查询订单.
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws InvalidArgumentException|ServiceNotFoundException
     */
    public function find(PayOrder $order) : array
    {
        return [
            'amount' => $order->amount,
            //预收金额
            'payer_total' => $this->calculate_amount($order->payer_total),
            //实收金额
            'status' => $order->status,
            // 订单状态
            'amount_total' => $this->calculate_amount($order->amount_total, true),
            // 订单总金额
            'success_time' => null,
        ];
    }
    /**
     * 关闭订单.
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws ServiceNotFoundException
     */
    public function close(PayOrder $order) : array
    {
        PayOrder::query()->where('id', $order->id)->update(['status' => '交易关闭']);
        return Json_Api(200, true, ['msg' => '订单已关闭!']);
    }
    /**
     * 取消订单.
     * @throws ContainerException
     * @throws InvalidParamsException
     * @throws ServiceNotFoundException
     */
    public function cancel(PayOrder $order) : array
    {
        PayOrder::query()->where('id', $order->id)->update(['status' => '订单取消']);
        return Json_Api(200, true, ['msg' => '取消订单成功!']);
    }
    /**
     * 计算实际金额.
     * @param int|string $amount
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
}