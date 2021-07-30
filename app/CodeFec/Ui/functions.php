<?php

namespace App\CodeFec\Ui;

class functions
{
    public static function Ui()
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(UiInterface::class);
    }

    public static function get($type)
    {
        $arr = [];
        foreach (\App\CodeFec\Ui\functions::Ui()->get() as $value) {
            if($value['type']==$type){
                $arr[]=$value["value"];
            }
        }
        return $arr;
    }
}
