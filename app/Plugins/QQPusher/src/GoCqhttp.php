<?php

namespace App\Plugins\QQPusher\src;

use App\Model\AdminOption;

class GoCqhttp
{
    public static function post($action='/get_version_info',$data=[]){
        try {
            return http()->post(self::get_options('QQPusher_POST_URL','http://127.0.0.1:5700')."/".$action,$data);
        }catch (\Exception $e){
            return ['status' => 'error'];
        }
    }

    public static function get_options($name,$default=""){
        if(!cache()->has('admin.options.'.$name)){
            cache()->set("admin.options.".$name,@AdminOption::query()->where("name",$name)->first()->value);
        }
        return self::core_default(cache()->get("admin.options.".$name),$default);
    }

    public static function core_default($string=null,$default=null){
        if($string){
            return $string;
        }
        return $default;
    }
}