<?php

namespace App\Plugins\User\src\Event;

// 登陆成功
class AfterLogin
{
    /**
     * 用户信息
     * @var array|object
     */
    public array|object $user;
    public function __construct($user)
    {
        $this->user = $user;
    }
}