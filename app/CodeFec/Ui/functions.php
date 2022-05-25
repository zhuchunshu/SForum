<?php

namespace App\CodeFec\Ui;

class functions
{
    public static function Ui()
    {
	    return \Hyperf\Utils\ApplicationContext::getContainer()->get(UiInterface::class);
    }

    public static function get($type): array
    {
        $arr = [];
        foreach (self::Ui()->get() as $value) {
            if($value['type']==$type){
                $arr[]=$value["value"];
            }
        }
        return $arr;
    }
}
