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
        if(Arr::has($this->list,$class)){
            $array = $this->list[$class];
            if($array && is_array($array)){
                ksort($array);
            }else{
                $array = [];
            }
            return $array;
        }
        return [];
    }
}