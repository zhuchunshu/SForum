<?php


namespace App\CodeFec\Itf\Itf;


interface ItfInterface
{
    /**
     * @param $class
     * @return array
     */
    public function get($class): array;

    /**
     * @param $class
     * @param $id
     * @param $data
     */
    public function add($class, $id,$data);

}