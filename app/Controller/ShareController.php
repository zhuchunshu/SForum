<?php

namespace App\Controller;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix:"/share")]
class ShareController
{
    #[GetMapping(path:"admin/server/logger/{_token}.debug")]
    public function admin_server_logger($_token){
        $data = admin_log()->db()->createQueryBuilder()->where(['_token','=',$_token])->getQuery()->first();
        return view('share.admin_server_logger',['data' => $data]);
    }
}