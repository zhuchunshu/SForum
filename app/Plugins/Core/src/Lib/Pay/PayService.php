<?php

namespace App\Plugins\Core\src\Lib\Pay;

use App\CodeFec\Ui\Generate\PayGenerate;
use App\Plugins\Core\src\Lib\Pay\Event\NotifyEvent;
use App\Plugins\Core\src\Lib\Pay\Jobs\Service\OrderCloseJobService;
use App\Plugins\Core\src\Models\PayConfig;
use App\Plugins\Core\src\Models\PayOrder;
use App\Plugins\User\src\Models\User;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;
use Psr\Http\Message\ResponseInterface;

class PayService
{

    /**
     * 关闭订单服务
     * @Inject
     * @var OrderCloseJobService
     */
    protected OrderCloseJobService $OrderCloseJobService;

    /**
     * @param string|int $user_id 用户id
     * @param string $title 订单标题
     * @param string $total_amount 订单金额
     * @param array $payment_method 支付方式
     * @return bool|array
     */
    public function create(string|int $user_id, string $title, string $total_amount, array $payment_method): bool|array
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
        if ($this->check_payment($payment_method) !== true) {
            return $this->check_payment($payment_method);
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
        // 定时关闭订单
        $this->OrderCloseJobService->push(['order_id' => $order->id, 'check_payment' => false, 'payServer' => $payServer]);
        $payServerHandler = $payServer['handler'];
        if (!@method_exists(new $payServerHandler(), 'create')) {
            return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单创建方法"]);
        }
        return (new $payServerHandler())->create($order);
    }

    /**
     * 获取所有支付接口
     * @return array
     */
    public function getInterfaces(): array
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
                $data[$this->core_Itf_id('Pay', $key)] = $Pays;
            }
        }
        return $data;
    }

    private function core_Itf_id($name, $id)
    {
        return \Hyperf\Utils\Str::after($id, $name . "_");
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
        return $this->core_default(cache()->get("admin.options.pay." . $name), $default);
    }

    private function core_default($string = null, $default = null)
    {
        if ($string) {
            return $string;
        }
        return $default;
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
     * 获取已启动支付插件
     * @return array
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function get_enabled_data(): array
    {
        // 获取在数据库中已启用的支付插件
        $pays = json_decode(pay()->get_options('enable', '[]'));
        $enable = [];
        foreach ($pays as $name) {
            foreach ($this->getInterfaces() as $id => $p) {
                if (Arr::has($p, 'ename') && $p['ename'] === $name) {
                    $enable[$id] = $p;
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
     * @param array $notify_result 回调数据
     * @param string $amount_total 总金额
     * @param string|null $payment_method
     * @return bool|array
     */
    public function notify(string $id, string $status, string $trade_no, string $payer_total, array $notify_result, string $amount_total, string|null $payment_method = null): bool|array
    {
        if (!PayOrder::query()->where('id', $id)->exists()) {
            return Json_Api(403, false, ['msg' => '订单号不存在']);
        }
        // 更新数据
        PayOrder::query()->where('id', $id)->update([
            'status' => $status,
            'trade_no' => $trade_no,
            'amount_total' => $amount_total,
            'payer_total' => $payer_total,
            'notify_result' => json_encode($notify_result, JSON_UNESCAPED_UNICODE, JSON_PRETTY_PRINT),
            'payment_method' => $payment_method ?: PayOrder::query()->find($id)->payment_method
        ]);
        // 触发事件
        EventDispatcher()->dispatch(new NotifyEvent($id));
        // 响应结果
        return true;
    }


    /**
     * 获取支付插件信息
     * @param $id
     * @param $ename
     * @return array|mixed
     */
    public function get_pay_plugin_data($id, $ename): mixed
    {
        if (!array_key_exists($id, $this->getInterfaces())) {
            return '支付方式不存在,id检索失败';
        }
        // 检索支付方式ename
        if (!Arr::has($this->getInterfaces()[$id], 'ename') || $this->getInterfaces()[$id]['ename'] !== $ename) {
            return '支付方式不存在,ename检索失败';
        }
        return $this->getInterfaces()[$id];
    }

    /**
     * 查询订单
     * @param $trade_no
     * @param bool $refund
     * @return array|ResponseInterface
     */
    public function find($trade_no, bool $refund = false): array|ResponseInterface
    {
        $order = PayOrder::query()->where('trade_no', $trade_no)->first();
        $payment_method = json_decode($order->payment_method, true);
        // 检索支付方式
        if ($this->check_payment($payment_method) !== true) {
            return $this->check_payment($payment_method);
        }
        // 支付插件信息
        $payServer = $this->get_ename_Interfaces()[$payment_method[1]];
        $payServerHandler = $payServer['handler'];
        if (!@method_exists(new $payServerHandler(), 'find')) {
            return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单查询方法"]);
        }
        $result = (new $payServerHandler())->find($order, $refund);
        return view('App::Pay.admin.order_show', ['order' => $order, 'result' => $result]);
    }

    /**
     * 生成前端html代码
     * @return PayGenerate
     */
    public function generate_html(): PayGenerate
    {
        return (new PayGenerate());
    }


    /**
     * 关闭订单
     * @param $id
     * @param bool $check_payment
     * @param null $payServer
     * @return array|ResponseInterface
     */
    public function close($id, $check_payment = true, $payServer = null)
    {
        $order = PayOrder::query()->find($id);
        if(Str::is('*待支付*','*'.$order->status.'*') || Str::is('*未支付*','*'.$order->status.'*') || Str::is('*未付款*','*'.$order->status.'*')){
            if ($order->trade_no) {
                PayOrder::query()->where('id', $order->id)->update([
                    'status' => '交易关闭'
                ]);
                return Json_Api(200, true, ['msg' => '交易已关闭!']);
            }
            PayOrder::query()->where('id', $order->id)->update([
                'status' => '交易关闭'
            ]);
            $payment_method = json_decode($order->payment_method, true);
            // 检索支付方式
            if (($check_payment === true) && $this->check_payment($payment_method) !== true) {
                return $this->check_payment($payment_method);
            }
            // 支付插件信息
            if ($payServer === null) {
                $payServer = $this->get_ename_Interfaces()[$payment_method[1]];
            }
            $payServerHandler = $payServer['handler'];
            if (!@method_exists(new $payServerHandler(), 'close')) {
                return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单查询方法"]);
            }
            return (new $payServerHandler())->close($order);
        }

        return Json_Api(403, false, ['msg' => '当前订单状态不允许关闭']);
    }

    /**
     * 取消订单
     * @param $id
     * @return array|ResponseInterface
     */
    public function cancel($id, $check_payment = true, $payServer = null)
    {
        $order = PayOrder::query()->find($id);
        if(Str::is('*待支付*','*'.$order->status.'*') || Str::is('*未支付*','*'.$order->status.'*') || Str::is('*未付款*','*'.$order->status.'*')) {
            if (!$order->trade_no) {
                PayOrder::query()->where('id', $order->id)->update([
                    'status' => '订单取消'
                ]);
                return Json_Api(200, true, ['msg' => '取消订单成功!']);
            }
            $payment_method = json_decode($order->payment_method, true);
            // 检索支付方式
            if (($check_payment === true) && $this->check_payment($payment_method) !== true) {
                return $this->check_payment($payment_method);
            }
            // 支付插件信息
            if ($payServer === null) {
                $payServer = $this->get_ename_Interfaces()[$payment_method[1]];
            }
            $payServerHandler = $payServer['handler'];
            if (!@method_exists(new $payServerHandler(), 'cancel')) {
                return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单查询方法"]);
            }
            return (new $payServerHandler())->cancel($order);
        }
        return Json_Api(403, false, ['msg' => '当前订单状态不允许取消']);
    }

    /**
     * 检索支付方式
     * @param array $payment_method
     * @return bool|array
     */
    private function check_payment(array $payment_method): bool|array
    {
        // 检索支付方式
        if (!is_array($payment_method) || !is_numeric($payment_method[0]) || !is_string($payment_method[1])) {
            return Json_Api(403, false, ['msg' => '支付方式格式不正确']);
        }
        // 检索支付方式id
        if (!Arr::has($this->getInterfaces(), $payment_method[0])) {
            return Json_Api(403, false, ['msg' => '支付方式不存在,id检索失败']);
        }
        // 检索支付方式ename
        if (!Arr::has($this->getInterfaces()[$payment_method[0]], 'ename') || $this->getInterfaces()[$payment_method[0]]['ename'] !== $payment_method[1]) {
            return Json_Api(403, false, ['msg' => '支付方式不存在,ename检索失败']);
        }

        // 检索支付方式id
        if (!is_array($this->get_enabled_data()[$payment_method[0]])) {
            return Json_Api(403, false, ['msg' => '此支付扩展未启用']);
        }

        return true;
    }

    /**
     * 对未付款订单进行付款
     * @param $order_id
     * @param $payment
     * @return array|bool|mixed
     */
    public function paying($order_id, $payment = null): mixed
    {
        if (!PayOrder::query()->where('id', $order_id)->exists()) {
            return Json_Api(403, false, ['msg' => '订单不存在']);
        }
        // 获取订单信息
        $order = PayOrder::query()->find($order_id);
        // 判断订单是否为未支付状态
        if ($order->status !== '待支付' && $order->status !== '待付款' && $order->status !== '未支付' && $order->status !== '未付款') {
            return Json_Api(403, false, ['msg' => '订单不为待支付状态']);
        }
        // 获取支付方式
        $payment_method = json_decode($order->payment_method);
        if ($payment) {
            $payment_method = json_decode($payment, true);
        }
        // 检索支付方式
        if ($this->check_payment($payment_method) !== true) {
            return $this->check_payment($payment_method);
        }
        // 修改数据库中的支付方式
        $order->payment_method = json_encode($payment_method, JSON_UNESCAPED_UNICODE);
        $order->save();
        // 支付插件信息
        $payServer = $this->get_ename_Interfaces()[$payment_method[1]];
        $payServerHandler = $payServer['handler'];
        if (!@method_exists(new $payServerHandler(), 'create')) {
            return Json_Api(500, false, ['msg' => '支付插件:' . $payment_method[1] . "无有效的订单创建方法"]);
        }
        return (new $payServerHandler())->create($order);
    }
}
