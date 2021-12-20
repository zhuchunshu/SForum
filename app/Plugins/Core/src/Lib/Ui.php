<?php

namespace App\Plugins\Core\src\Lib;

use App\Plugins\Core\src\Lib\Ui\Css;
use App\Plugins\Core\src\Lib\Ui\Html;

class Ui
{
    public function Css(): Css
    {
        return new Css();
    }

    public function Html(): Html
    {
        return new Html();
    }
}