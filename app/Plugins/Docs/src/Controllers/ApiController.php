<?php

namespace App\Plugins\Docs\src\Controllers;

use App\Plugins\Docs\src\Model\Docs;
use App\Plugins\Docs\src\Model\DocsClass;
use App\Plugins\Docs\src\Request\CreateDocsRequest;
use App\Plugins\Docs\src\Request\EditDocsRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix:"/api/docs")]
class ApiController
{
    ##[PostMapping(path:"getDocs")]
    public function getDocs(){
        $class_id = request()->input('id');
        if(!$class_id){
            return Json_Api(403,false,['msg'=>'请求参数不足,缺少:id']);
        }
        if(!DocsClass::query()->where('id',$class_id)->exists()){
            return Json_Api(403,false,['msg'=>'ID为'.$class_id.'的文档不存在']);
        }
        if(!Docs::query()->where("class_id",$class_id)->exists()){
            return Json_Api(403,false,['msg'=>'ID为'.$class_id.'分类下的文档不存在']);
        }
        $classData = DocsClass::query()->where('id',$class_id)->with('user')->first();
        $page = Docs::query()->where("class_id",$class_id)->get();
        return Json_Api(200,true,['msg' => '加载成功!','classData' => $classData,'docs' => $page]);
    }

    // 发布文档
    #[PostMapping(path:"create")]
    public function create(CreateDocsRequest $request){
        $title = $request->input("title");
        $class_id = $request->input("class_id");
        $user_id = (int)DocsClass::query()->where('id',$class_id)->first()->user_id;
        if(auth()->id()!==$user_id || !Authority()->check("docs_create")){
            return Json_Api(403,false,['无权限!']);
        }
        $markdown = $request->input("markdown");
        $html = $request->input("html");
        $html = xss()->clean($html);
        // 解析标签
        $html = $this->tag($html);
        // 解析艾特
        $html = $this->at($html);
        Docs::query()->create([
            'user_id' => auth()->id(),
            'class_id' => $class_id,
            'title' => $title,
            'content' => $html,
            'markdown' => $markdown
        ]);
        return Json_Api(200,true,['发布成功!']);
    }

    // 修改文档
    #[PostMapping(path:"edit")]
    public function edit(EditDocsRequest $request){
        $title = $request->input("title");
        $user_id = (int)Docs::query()->where('id',$request->input('id'))->first()->user_id;
        $quanxian = false;
        if(auth()->id()===$user_id && Authority()->check("docs_edit")){
            $quanxian = true;
        }

        if(Authority()->check("admin_docs_edit")){
            $quanxian = true;
        }

        if(!$quanxian){
            return Json_Api(403,false,['无权限!']);
        }
        $markdown = $request->input("markdown");
        $html = $request->input("html");
        $html = xss()->clean($html);
        // 解析标签
        $html = $this->tag($html);
        // 解析艾特
        $html = $this->at($html);
        Docs::query()->where("id",$request->input('id'))->update([
            'title' => $title,
            'content' => $html,
            'markdown' => $markdown
        ]);
        return Json_Api(200,true,['更新成功!']);
    }

    public function tag(string $html): string
    {
        return replace_all_keywords($html);
    }

    public function at(string $html): string
    {
        return replace_all_at($html);
    }

    // 删除文档分类
    #[PostMapping(path:"classDelete")]
    public function deleteDocsClass(){
        if(!request()->input("class_id")){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:class_id']);
        }
        $quanxian = false;
        $data = DocsClass::query()->where("id",request()->input("class_id"))->first();
        if(auth()->id()===(int)$data->user_id && Authority()->check("docs_delete")){
            $quanxian = true;
        }
        if(Authority()->check("admin_docs_delete")){
            $quanxian = true;
        }
        if($quanxian===false){
            return Json_Api(401,false,['msg' => '无权限!']);
        }
        Docs::query()->where("class_id",$data->id)->delete();
        DocsClass::query()->where("id",$data->id)->delete();
        return Json_Api(200,true,['msg' => '删除成功!']);
    }

    // 删除文档
    #[PostMapping(path:"docsDelete")]
    public function deleteDocs(){
        if(!request()->input("id")){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
        }
        $quanxian = false;
        $data = Docs::query()->where("id",request()->input("id"))->first();
        if(auth()->id()===(int)$data->user_id && Authority()->check("docs_delete")){
            $quanxian = true;
        }
        if(Authority()->check("admin_docs_delete")){
            $quanxian = true;
        }
        if($quanxian===false){
            return Json_Api(401,false,['msg' => '无权限!']);
        }
        Docs::query()->where("id",request()->input("id"))->delete();
        return Json_Api(200,true,['msg' => '删除成功!']);
    }

    #[PostMapping(path:"editDocsData")]
    public function editDocsData()
    {
        $id = request()->input("id");
        if(!$id){
            return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
        }
        if(!Docs::query()->where("id",$id)->exists()){
            return Json_Api(404,false,['msg' => '文档不存在']);
        }
        $data = Docs::query()->with("docsClass")->where("id",$id)->first();
        $quanxian = false;
        if((int)$data->user_id === auth()->id() && Authority()->check("docs_edit")){
            $quanxian = true;
        }
        if(Authority()->check("admin_docs_edit")){
            $quanxian = true;
        }
        if(!$quanxian){
            return Json_Api(401,false,['msg' => '无权查看']);
        }
        return Json_Api(200,true,$data);
    }
}