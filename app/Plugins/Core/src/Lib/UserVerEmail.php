<?php

namespace App\Plugins\Core\src\Lib;

class UserVerEmail
{
    public function make(){
        $id = auth()->data()->id;
        cache()->set("core.user.ver.email.".$id,\Hyperf\Utils\Str::random(7),1800);
        cache()->set("core.user.ver.email.time.".$id,time()+60);
        return cache()->get("core.user.ver.email.".$id);
    }

    public function check($captcha): bool
    {
        $id = auth()->data()->id;
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
        $id = auth()->id();
        if(!cache()->has("core.user.ver.email.time.".$id)){
            return true;
        }
        if(cache()->get("core.user.ver.email.time.".$id,0)-time()<=0){
            return true;
        }
        return false;
    }

    public function sendTime(): int{
        $id = auth()->id();
        return cache()->get("core.user.ver.email.time.".$id,0)-time();
    }

    public function send($email){
        $mail = Email();
        $user = auth()->data();
        $captcha = $this->make();
        $mail->addAddress($email);
        $mail->Subject = "【".get_options("web_name")."】请查看你的邮箱验证码";
        $mail->Body    = <<<HTML
你好 {$user->username},<br>
你的邮箱验证码是:{$captcha}
HTML;
        $mail->send();
        return $captcha;
    }
}