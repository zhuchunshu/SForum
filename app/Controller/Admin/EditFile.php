<?php

namespace App\Controller\Admin;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller]
#[Middleware(\App\Middleware\AdminMiddleware::class)]
class EditFile
{
    #[GetMapping(path:"/admin/EditFile/css")]
    public function css(): \Psr\Http\Message\ResponseInterface
    {
        $content = $this->getFile(public_path("css/diy.css"));
        $action = "/admin/EditFile/css";
        return view("admin.EditFile",["content"=>$content,"lang" => "css","action" => $action,"title" => "自定义CSS代码"]);
    }

    #[PostMapping(path:"/admin/EditFile/css")]
    public function css_save(){
        if(!admin_auth()->check()){
            return Json_Api(401,false,['msg' => '无权限']);
        }
        $content = request()->input("content");
        if($this->PutFile(public_path("css/diy.css"),$content)){
            return Json_Api(200,true,['msg' => '修改成功!']);
        }

        return Json_Api(200,false,['msg' => '修改失败!']);
    }

    #[PostMapping(path:"/admin/EditFile/js")]
    public function js_save(){
        if(!admin_auth()->check()){
            return Json_Api(401,false,['msg' => '无权限']);
        }
        $content = request()->input("content");
        if($this->PutFile(public_path("js/diy.js"),$content)){
            return Json_Api(200,true,['msg' => '修改成功!']);
        }

        return Json_Api(200,false,['msg' => '修改失败!']);
    }

    #[GetMapping(path:"/admin/EditFile/js")]
    public function js(): \Psr\Http\Message\ResponseInterface
    {
        $content = $this->getFile(public_path("js/diy.js"));
        $action = "/admin/EditFile/js";
        return view("admin.EditFile",["content"=>$content,"lang" => "javascript","action" => $action,"title" => "自定义JS代码"]);
    }

    public function PutFile($path,$content=null): bool
    {
        if(!$content){
            return false;
        }
        file_put_contents($path,$content);
        return true;
    }

    public function getFile($path,$content=" "): bool|string
    {
        if(!file_exists($path)){
            file_put_contents($path,$content);
        }
        return file_get_contents($path);
    }
}