<?php

namespace App\Plugins\Core\src\Lib\Sms;


class Sms
{
    public function send($to,$data=[]): bool
    {
        if(get_options("core_user_sms","关闭")==="开启"){
            $sms_service = get_options("core_user_sms_service",'Qcloud');

            $handler = null;
            foreach (Itf()->get('SMS') as $value){
                if($value['name']===$sms_service){
                    $handler = $value['handler'];
                }
            }
            if($handler){
                (new $handler)->handler($to,$data);
            }
            return false;
        }
        return false;
    }
}