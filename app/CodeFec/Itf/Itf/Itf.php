<?php


namespace App\CodeFec\Itf\Itf;


use Illuminate\Support\Arr;

class Itf implements ItfInterface
{

    public $list=[];

    public function add($class, $id,$data): bool
    {
        $this->list = Arr::add($this->list, $class.".".$class."_".$id, $data);
        return true;
    }

    public function get($class): array
    {
        $array = $this->list[$class];
        ksort($array);
        return $array;
    }
}