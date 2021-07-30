<?php

namespace App\Plugins\Core\src\Lib;


use Illuminate\Support\Str;
use Psr\SimpleCache\InvalidArgumentException;

class Captcha
{
    public function get(){
        if(!session()->has("core_captcha")){
            session()->set("core_captcha",Str::random(15));
        }
        if(!cache()->has("core.captcha.".session()->get("core_captcha"))){
            $data = [
                "add1" => mt_rand(12,50),
                "add2" => mt_rand(3,12),
                "time" => time()+600,
            ];
            cache()->set("core.captcha.".session()->get("core_captcha"),$data,600);
        }
        return cache()->get("core.captcha.".session()->get("core_captcha"));
    }


    public function reget(){
        if(!session()->has("core_captcha")){
            session()->set("core_captcha",Str::random(15));
        }
        cache()->delete("core.captcha.".session()->get("core_captcha"));
        if(!cache()->has("core.captcha.".session()->get("core_captcha"))){
            $data = [
                "add1" => mt_rand(12,50),
                "add2" => mt_rand(3,12),
                "time" => time()+600,
            ];
            cache()->set("core.captcha.".session()->get("core_captcha"),$data,600);
        }
        return cache()->get("core.captcha.".session()->get("core_captcha"));
    }

    /**
     * @param int $value
     * @return bool
     */
    public function validate($value)
    {
        if(!session()->has("core_captcha")){
            return false;
        }
        if(!cache()->has("core.captcha.".session()->get("core_captcha"))){
            return false;
        }
        $data = cache()->get("core.captcha.".session()->get("core_captcha"));
        $add1 = $data['add1'];
        $add2  = $data['add2'];

        $result = ($add1+$add2);
        if($result == $value){
            return true;
        }else{
            return false;
        }
    }
}