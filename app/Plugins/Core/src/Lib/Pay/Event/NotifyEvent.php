<?php

namespace App\Plugins\Core\src\Lib\Pay\Event;

class NotifyEvent
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