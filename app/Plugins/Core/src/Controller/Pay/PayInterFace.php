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

use App\Plugins\Core\src\Models\PayOrder;

interface PayInterFace
{
    /**
     * 创建订单.
     * @param PayOrder $order 订单
     * @return mixed
     */
    public function create(PayOrder $order): mixed;

    /**
     * 支付回调.
     * @param mixed $mixed
     */
    public function notify(mixed $mixed): mixed;

    /**
     * 查询订单.
     * @return mixed
     */
    public function find(PayOrder $order): array;

    /**
     * 关闭订单.
     */
    public function close(PayOrder $order): array;

    /**
     * 取消订单.
     */
    public function cancel(PayOrder $order): array;
}
