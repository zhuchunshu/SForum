<?php

namespace App\Plugins\Core\src\Lib\Xss;

use HTMLPurifier;
use HTMLPurifier_Config;
use HTMLPurifier_HTML5Config;

class Xss
{
    public \HTMLPurifier_Config $config;

    public function __construct(){
        $this->config();
    }

    public function config(): void
    {
        $config =HTMLPurifier_HTML5Config::createDefault();
        $this->config = $config;
    }

    public function clean($html): string
    {
        $purifier = new HTMLPurifier($this->config);
        return $purifier->purify($html);
    }

}