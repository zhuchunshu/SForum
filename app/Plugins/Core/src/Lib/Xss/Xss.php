<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib\Xss;

use HTMLPurifier;
use HTMLPurifier_HTML5Config;

class Xss
{
    public \HTMLPurifier_Config $config;

    public function __construct()
    {
        $this->config();
    }

    public function config(): void
    {
        $config = HTMLPurifier_HTML5Config::createDefault();
        $config->set('CSS.AllowTricky', true);

        $this->config = $config;
    }

    public function clean($html)
    {
        $purifier = new HTMLPurifier($this->config);
        return $purifier->purify($html);
    }
}
