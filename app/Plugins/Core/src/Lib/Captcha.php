<?php

namespace App\Plugins\Core\src\Lib;


use Illuminate\Support\Str;

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
                "time" => time()+300,
            ];
            cache()->set("core.captcha.".session()->get("core_captcha"),$data,300);
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
                "time" => time()+300,
            ];
            cache()->set("core.captcha.".session()->get("core_captcha"),$data,300);
        }
        return cache()->get("core.captcha.".session()->get("core_captcha"));
    }

    /**
     * @param int $value
     */
    public function validate(int $value){

    }
}