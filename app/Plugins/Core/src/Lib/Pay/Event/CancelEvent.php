<?php

namespace App\Plugins\Core\src\Lib\Pay\Event;

/**
 *
 * 交易取消事件
 */
class CancelEvent
{
    /**
     * 订单id
     * @var int|string
     */
    public $id;

    public function __construct($id){
        $this->id = $id;
    }
}