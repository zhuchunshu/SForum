<?php
namespace App\CodeFec\Itf\Setting;

use Hyperf\Collection\Arr;

class Setting implements SettingInterface {

    public $list=[];

    public function get(): array
    {
        $array = $this->list;
        ksort($array);
        return $array;
    }

    public function add(int $id, string $name,string $ename,string $view): bool
    {

        $arr = [
            'name' => $name,
            'ename' => $ename,
            'view' => $view
        ];
        $this->list = Arr::set($this->list, $id, $arr);

        return true;
    }

}