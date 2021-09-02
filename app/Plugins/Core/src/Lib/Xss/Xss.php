<?php

namespace App\Plugins\Core\src\Lib\Xss;

use HTMLPurifier;
use HTMLPurifier_Config;

class Xss
{
    public \HTMLPurifier_Config $config;

    public function __construct(){
        $this->config();
    }

    public function config(): void
    {
        $this->config = HTMLPurifier_Config::createDefault();
    }

    public function clean($html): string
    {
        $purifier = new HTMLPurifier($this->config);
        return $purifier->purify($html);
    }
}