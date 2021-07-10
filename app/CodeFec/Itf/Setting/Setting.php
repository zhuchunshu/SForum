<?php
namespace App\CodeFec\Itf\Setting;

use Illuminate\Support\Arr;

class Setting implements SettingInterface {

    public static $list=[];

    public static function get(): array
    {
        return self::$list;
    }

    public static function add(int $id, string $name,string $ename,string $view): bool
    {

        $arr = [
            'name' => $name,
            'ename' => $ename,
            'view' => $view
        ];
        self::$list = Arr::add(self::$list, $id, $arr);

        return true;
    }

}