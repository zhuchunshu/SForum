<?php

namespace App\Plugins\Core\src\Lib\Ui;

class Css
{
    public function bg_color($color){
        return <<<CSS
background:{$color}
CSS;

    }

    public function bg_color_lt($color){
        return <<<CSS
color: {$color}!important;
background: rgba(32,107,196,.1)!important;
CSS;

    }
}