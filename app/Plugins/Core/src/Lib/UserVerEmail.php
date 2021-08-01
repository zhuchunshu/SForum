<?php

namespace App\Plugins\Core\src\Lib;

class UserVerEmail
{
    public function make(){
        $id = auth()->id;
        cache()->set("core.user.ver.email.".$id,\Hyperf\Utils\Str::random(7));
        cache()->set("core.user.ver.email.time.".$id,time()+60);
        return cache()->get("core.user.ver.email.".$id);
    }

    public function check($captcha): bool
    {
        $id = auth()->id;
        if(!cache()->has("core.user.ver.email.".$id)){
            return false;
        }
        $data = cache()->get("core.user.ver.email.".$id);
        if($captcha===$data){
            return true;
        }
        return false;
    }

    public function ifsend(): bool
    {
        $id = auth()->id;
        if(!cache()->has("core.user.ver.email.time.".$id)){
            return true;
        }
        if(cache()->get("core.user.ver.email.time.".$id,0)-time()<=0){
            return true;
        }
        return false;
    }

    public function sendTime(): int{
        $id = auth()->id;
        return cache()->get("core.user.ver.email.time.".$id,0)-time();
    }
}