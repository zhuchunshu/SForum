<?php
namespace App\CodeFec;

class Plugins {

    public static function GetAll(): array
    {
        $arr = getPath(plugin_path());
        $plugin_arr = [];
        foreach ($arr as $value) {
            if(file_exists(plugin_path($value."/".$value.".php"))){
                $plugin_arr[$value]['dir']=$value;
                $plugin_arr[$value]['path']=plugin_path($value);
                $plugin_arr[$value]['class']="\App\Plugins\\".$value."\\".$value;
                $plugin_arr[$value]['data']=get_plugins_doc($plugin_arr[$value]['class']);
                $plugin_arr[$value]['file']=plugin_path($value."/".$value.".php");
            }
        }
        return $plugin_arr;
    }

}