<?php
namespace App\CodeFec\Itf\Setting;

use Illuminate\Support\Arr;

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
        $this->list = Arr::add($this->list, $id, $arr);

        return true;
    }

}