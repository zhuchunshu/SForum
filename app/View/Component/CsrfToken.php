<?php

namespace App\View\Component;

use Hyperf\ViewEngine\Component\Component;
use function Hyperf\ViewEngine\view as views;

class CsrfToken extends Component
{
    public mixed $token;
    public function __construct()
    {
        $this->token = csrf_token();
    }

    public function render(): mixed
    {
        return views('components.csrf');
    }
}
