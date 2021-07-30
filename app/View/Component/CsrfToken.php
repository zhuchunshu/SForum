<?php

namespace App\View\Component;

use Hyperf\ViewEngine\Component\Component;
use function Hyperf\ViewEngine\view as views;

class CsrfToken extends Component
{
    public $token;
    public function __construct()
    {
        $this->token = recsrf_token();
    }
    public function render()
    {
        return views('components.csrf');
    }
}
