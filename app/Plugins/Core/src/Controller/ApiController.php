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

        // 鉴权
        $quanxian = false;
        if(($request->input("type") === "comment") && Authority()->check("report_comment")) {
            $quanxian = true;
        }
        if(($request->input("type") === "topic") && Authority()->check("report_topic")) {
            $quanxian = true;
        }

        if($quanxian===false){
            return Json_Api(401,false,['无权限']);
        }

        //
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

    // 获取举报信息
    #[PostMapping(path:"report/data")]
    public function report_data(){
        $report_id = request()->input('report_id');
        if(!$report_id){
            return Json_Api(403,false,['请求参数不足,缺少:report_id']);
        }
        if(!Report::query()->where("id",$report_id)->exists()){
            return Json_Api(403,false,['id为'.$report_id."的举报内容不存在"]);
        }
        $data = Report::query()->where("id",$report_id)->first(['status','type','_id']);
        return Json_Api(200,true,$data);
    }

    #[PostMapping(path:"report/update")]
    public function report_update(){
        if(!auth()->check() || !Authority()->check("admin_report")){
            return Json_Api(401,false,['无权限']);
        }
        $report_id = request()->input('report_id');
        if(!$report_id){
            return Json_Api(403,false,['请求参数不足,缺少:report_id']);
        }
        if(!Report::query()->where("id",$report_id)->exists()){
            return Json_Api(403,false,['id为'.$report_id."的举报内容不存在"]);
        }
        $status = Report::query()->where("id",$report_id)->first(['status'])->status;
        if($status==="pending"){
            $_status="approve";
            $_text="批准";
        }
        if($status==="reject"){
            $_status="approve";
            $_text="批准";
        }
        if($status==="approve"){
            $_status="reject";
            $_text="驳回";
        }
        Report::query()->where("id",$report_id)->update([
            'status' => $_status
        ]);
        return Json_Api(200,true,[$_text."成功!"]);
    }

    // 删除举报
    #[PostMapping(path:"report/remove")]
    public function report_remove(){
        if(!auth()->check() || !Authority()->check("admin_report")){
            return Json_Api(401,false,['无权限']);
        }
        $report_id = request()->input('report_id');
        if(!$report_id){
            return Json_Api(403,false,['请求参数不足,缺少:report_id']);
        }
        if(!Report::query()->where("id",$report_id)->exists()){
            return Json_Api(403,false,['id为'.$report_id."的举报内容不存在"]);
        }

        // 举报快照
        $data = Report::query()->where("id",$report_id)->first();

        Report::query()->where("id",$report_id)->delete();


        // 发送通知
        $users = [];
        foreach (Authority()->getUsers("admin_report") as $user){
            $users[]=$user->id;
        }

        $user_data = auth()->data();
        $mail_content = view("Core::report.remove_admin",['data' => $data,'user' => $user_data]);

        user_notice()->sends($users,"有管理员删除了一条举报,特此通知!",$mail_content,url("/"));
        return Json_Api(200,true,['删除成功!']);
    }

    // 获取所有被举报并批准的的评论
    #[PostMapping(path:"report/approve.comment")]
    public function report_approve_comment_list(){
        $arr = [];
        foreach (Report::query()->where(["type"=>"comment",'status' => 'approve'])->get() as $value){
            $arr[]=$value->_id;
        }
        return Json_Api(200,true,$arr);
    }
}