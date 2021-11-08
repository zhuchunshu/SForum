<?php

namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Models\Report;
use App\Plugins\Core\src\Request\Report\CreateRequest;
use App\Plugins\User\src\Lib\UserNotice;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller(prefix:"/api/core")]
#[RateLimit(create:1, capacity:3)]
class ApiController
{
    // 创建举报
    #[PostMapping(path:"report/create")]
    public function report_create(CreateRequest $request){
        if(!auth()->check()){
            return Json_Api(401,false,['未登录']);
        }
        $type = $request->input("type");
        $type_id = $request->input("type_id");
        if(Report::query()->where(['user_id' => auth()->id(),'_id' => $type_id,'type' => $type])->exists()){
            return Json_Api(403,false,['你已举报此贴,无需重复举报']);
        }
        $content = '**违规页面地址:** '.$request->input("url").'
**举报原因:** '.$request->input("report_reason")."\n\n".$request->input('content');
        $data = Report::query()->create([
            "type" => $type,
            "_id" => $type_id,
            "user_id" => auth()->id(),
            "title" => $request->input('title'),
            'content' => xss()->clean(markdown()->text($content))
        ]);

        // 发送通知
        $users = [];
        foreach (Authority()->getUsers("admin_report") as $user){
            $users[]=$user->id;
        }
        $mail_content = view("Core::report.send_admin",['data' => $data]);

        user_notice()->sends($users,"有用户举报了一条内容,需要你来审核",$mail_content,url("/report/".$data->id.".html"));
        return Json_Api(200,true,['举报成功! 等待管理员审核']);
    }
}