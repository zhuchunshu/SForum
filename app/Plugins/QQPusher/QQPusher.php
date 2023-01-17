<?php

namespace App\Plugins\QQPusher;

class QQPusher
{
    public function handler(){
        $this->bootstrap();
    }

    private function bootstrap(){
        require_once __DIR__."/bootstrap.php";
    }
}