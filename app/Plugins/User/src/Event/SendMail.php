<?php

namespace App\Plugins\User\src\Event;

class SendMail
{
    /**
     * 用户id
     * @var int|string
     */
    public int|string $user_id;

    /**
     * 标题
     * @var string
     */
    public string $title;

    /**
     * action 跳转链接
     * @var string|null
     */
    public null|string $action;
    
    public function __construct($user_id, $title, $action)
    {
        $this->user_id = $user_id;
        
        $this->title = $title;
        
        $this->action = $action;
    }
}
