<?php

namespace App\Plugins\Core\src\Lib\Pay\Event;

/**
 * 交易关闭事件
 */
class CloseEvent
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