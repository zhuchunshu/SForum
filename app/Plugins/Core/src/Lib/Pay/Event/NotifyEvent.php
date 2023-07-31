<?php

namespace App\Plugins\Core\src\Lib\Pay\Event;

/**
 * 回调通知事件
 */
class NotifyEvent
{
    /**
     * 订单id
     * @var int|string
     */
    public $id;
    public function __construct($id)
    {
        $this->id = $id;
    }
}