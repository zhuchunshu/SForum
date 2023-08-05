<?php

namespace App\Plugins\Core\src\Controller\Order;

use App\Plugins\Core\src\Models\PayOrder;
use App\Plugins\User\src\Middleware\AuthMiddleware;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Stringable\Str;
#[Controller(prefix: "/user/order")]
#[Middleware(LoginMiddleware::class)]
#[Middleware(AuthMiddleware::class)]
class UserController
{
    #[GetMapping("{id}.order")]
    public function order_show($id)
    {
        // 判断订单是否存在
        if (!PayOrder::query()->where('id', $id)->where('user_id', auth()->id())->exists()) {
            return admin_abort('订单不存在', 403);
        }
        // 订单信息
        $order = PayOrder::query()->find($id);
        return view('User::Order.show', ['order' => $order]);
    }
    #[PostMapping("{id}.order/paying")]
    public function order_paying($id)
    {
        // 判断订单是否存在
        if (!PayOrder::query()->where('id', $id)->where('user_id', auth()->id())->exists()) {
            return Json_Api(403, false, ['msg' => '订单不存在.' . $id]);
        }
        // 订单信息
        $order = PayOrder::query()->find($id);
        if (Str::is('*关闭*', $order->status)) {
            return Json_Api(403, false, ['msg' => '订单已关闭']);
        }
        if (Str::is('*取消*', $order->status)) {
            return Json_Api(403, false, ['msg' => '订单已取消']);
        }
        return pay()->paying($order->id, request()->input('payment'));
    }
    #[PostMapping("{id}.order/status")]
    public function get_order_status($id)
    {
        // 判断订单是否存在
        if (!PayOrder::query()->where('id', $id)->where('user_id', auth()->id())->exists()) {
            return Json_Api(403, false, ['msg' => '订单不存在.' . $id]);
        }
        // 订单信息
        $order = PayOrder::query()->find($id);
        $status = $order->status;
        if (Str::is('*成功*', $status)) {
            $status = '支付成功';
        }
        return Json_Api(200, true, ['msg' => '获取成功!', 'status' => $status]);
    }
}