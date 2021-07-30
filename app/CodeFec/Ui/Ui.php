<?php
namespace App\CodeFec\Ui;

use Illuminate\Support\Arr;

class Ui implements UiInterface {

    public $list =[
        // 0 =>[
        //     "type" => 0,
        //     "view" => "shared.header.left"
        // ],
        // 1 => [
        //     "type" => 1,
        //     "view" => "shared.header.right"
        // ]
    ];

    public function get(){
        return $this->list;
    }

    public function add(int $id, string $type,string $value)
    {
        $arr = [
            "type" => $type,
            "value" => $value,
        ];
        $this->list = Arr::add($this->list,$id,$arr);
        return true;
    }

}