<?php

namespace App\Plugins\Core\src\Lib\Pay;

use App\Plugins\Core\src\Lib\Pay\Event\NotifyEvent;
use App\Plugins\User\src\Models\User;
use App\Plugins\Core\src\Models\{PayConfig, PayOrder};
use Hyperf\Utils\{Arr, Str};

class PayService
{

    /**
     * @param string|int $user_id 用户id
     * @param string $title 订单标题
     * @param string $total_amount 订单金额
     * @param array $payment_method 支付方式
     * @return array
     */
    public function create(string|int $user_id, string $title, string $total_amount, array $payment_method)
    {
        // 判断用户是否存在
        if (!User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['msg' => '用户不存在']);
        }
        // 截取标题
        $title = Str::limit($title, 200);
        // 判断金额类型
        if (!is_numeric($total_amount)) {
            return Json_Api(403, false, ['msg' => '订单金额格式不正确']);
        }
        // 检索支付方式
        if (!is_array($payment_method) || !is_numeric($payment_method[0]) || !is_string($payment_method[1])) {
            return Json_Api(403, false, ['msg' => '支付方式格式不正确']);
        }
        // 检索支付方式id
        if (!array_key_exists($payment_method[0], $this->getInterfaces())) {
            return Json_Api(403, false, ['msg' => '支付方式不存在,id检索失败']);
        }
        // 检索支付方式ename
        if (!Arr::has($this->getInterfaces()[$payment_method[0]], 'ename') || $this->getInterfaces()[$payment_method[0]]['ename'] !== $payment_method[1]) {
            return Json_Api(403, false, ['msg' => '支付方式不存在,ename检索失败']);
        }
        // 创建订单
        $order = PayOrder::create([
            'id' => date("Ymd") . $user_id . date("His") . rand(1, 9) * rand(2, 11), // 订单号
            'title' => $title,
            'status' => '待支付',
            'user_id' => $user_id,
            'amount' => $total_amount,
            'payment_method' => json_encode($payment_method, JSON_UNESCAPED_UNICODE)
        ]);
        // 支付插件信息
        $payServer = $this->get_ename_Interfaces()[$payment_method[1]];
        $payServerHandler = $payServer['handler'];
        if (!@method_exists(new $payServerHandler(), 'create')) {
            return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单创建方法"]);
        }
        return (new $payServerHandler())->create($order);
    }

    public function pull()
    {

    }

    /**
     * 获取所有支付接口
     *
     */
    public function getInterfaces()
    {
        $data = [];
        foreach (Itf()->get('Pay') as $key => $Pays) {
            if (
                Arr::has($Pays, 'name')
                && Arr::has($Pays, 'description')
                && Arr::has($Pays, 'ename')
                && Arr::has($Pays, 'handler')
                && Arr::has($Pays, 'icon')
                && Arr::has($Pays, 'view')
            ) {
                $data[core_Itf_id('Pay', $key)] = $Pays;
            }
        }
        return $data;
    }

    /**
     * 获取所有支付接口
     *
     */
    public function get_ename_Interfaces()
    {
        $data = [];
        foreach (Itf()->get('Pay') as $key => $Pays) {
            if (
                Arr::has($Pays, 'name')
                && Arr::has($Pays, 'description')
                && Arr::has($Pays, 'ename')
                && Arr::has($Pays, 'handler')
                && Arr::has($Pays, 'icon')
                && Arr::has($Pays, 'view')
            ) {
                $data[$Pays['ename']] = $Pays;
            }
        }
        return $data;
    }

    /**
     * 获取配置内容
     * @param string $name
     * @param string $default
     * @return mixed|null
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function get_options(string $name, string $default = "")
    {
        if (!cache()->has('admin.options.pay.' . $name)) {
            cache()->set("admin.options.pay." . $name, @PayConfig::query()->where("name", $name)->first()->value);
        }
        return core_default(cache()->get("admin.options.pay." . $name), $default);
    }

    /**
     * 清理配置缓存
     * @return bool
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function clean_options(): bool
    {
        foreach (PayConfig::query()->get() as $value) {
            cache()->delete('admin.options.pay.' . $value->name);
        }
        return true;
    }

    /**
     * 获取已启动支付插件
     * @return array
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function get_enabled(): array
    {
        // 获取在数据库中已启用的支付插件
        $pays = json_decode(pay()->get_options('enable', '[]'));
        $enable = [];
        foreach ($pays as $name) {
            foreach ($this->getInterfaces() as $id => $p) {
                if (Arr::has($p, 'ename')) {
                    if ($p['ename'] === $name) {
                        $enable[] = $name;
                    }
                }
            }
        }
        return $enable;
    }

    /**
     * 处理回调通知
     * @param string $id 订单号(由super-forum系统生产)
     * @param string $status 订单状态
     * @param string $trade_no 交易单号
     * @param string $payer_total 实收金额
     * @param string $notify_result 回调数据
     * @param string $amount_total 总金额
     * @return bool|array
     */
    public function notify(string $id, string $status, string $trade_no, string $payer_total, array $notify_result, string $amount_total): bool|array
    {
        if (!PayOrder::query()->where('id', $id)->exists()) {
            return Json_Api(403, false, ['msg' => '订单号不存在']);
        }
        // 更新数据
        PayOrder::where('id', $id)->update([
            'status' => $status,
            'trade_no' => $trade_no,
            'amount_total' => $amount_total,
            'payer_total' => $payer_total,
            'notify_result' => json_encode($notify_result, JSON_UNESCAPED_UNICODE, JSON_PRETTY_PRINT),
        ]);
        // 触发事件
        EventDispatcher()->dispatch(new NotifyEvent($id));
        // 响应结果
        return true;
    }
}