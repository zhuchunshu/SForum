<?php

namespace App\Plugins\User\src\Event;

class SendNotice
{

    public string|int $user_id;

    public string $title;

    public string $content;

    public string $action;

    public function __construct($user_id,$title,$content,$action){
        $this->user_id = $user_id;
        $this->title = $title;
        $this->content = $content;
        $this->action = $action;
    }
}