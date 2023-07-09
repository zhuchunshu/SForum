<?php

namespace App\Plugins\User\src\Event\Task\System;

class SetAvatar
{
    public int $user_id;
    public function __construct(int $user_id)
    {
        $this->user_id = $user_id;
    }
}