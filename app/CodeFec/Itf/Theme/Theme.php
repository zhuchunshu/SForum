<?php


namespace App\CodeFec\Itf\Theme;


use Hyperf\Collection\Arr;

class Theme implements ThemeInterface
{

    public array $list=[];

    public function set($namespace, $hints)
    {
        $this->list[$namespace]=$hints;
    }

    public function get(): array
    {
        return $this->list;
    }
}