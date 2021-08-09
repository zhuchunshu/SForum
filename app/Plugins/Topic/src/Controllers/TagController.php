<?php


namespace App\Plugins\Topic\src\Controllers;


use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Requests\CreateTagRequest;
use App\Plugins\Topic\src\Requests\EditTagRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

#[Middleware(AdminMiddleware::class)]
#[Controller]
class TagController
{
    #[GetMapping(path:"/admin/topic/tag/create")]
    public function create(){
        return view("plugins.Topic.Tag.create");
    }

    #[PostMapping(path:"/admin/topic/tag/create")]
    public function create_store(CreateTagRequest $request,AvatarUpload $upload){
        $name = $request->input("name");
        $color = $request->input("color");
        $description = $request->input("description");
        $icon = $request->file("icon");
        $icon_url = $upload->save($icon,"admin",Str::random())['path'];
        TopicTag::create([
            "name" => $name,
            "color" => $color,
            "description" => $description,
            "icon" => $icon_url
        ]);
        return redirect()->url("/admin/topic/tag")->with("success","创建成功!")->go();
    }

    #[GetMapping(path:"/admin/topic/tag")]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->paginate(15);
        return view("plugins.Topic.Tag.index",["page" => $page]);
    }

    #[GetMapping(path:"/admin/topic/tag/edit/{id}")]
    public function edit($id){
        if(!TopicTag::query()->where("id",$id)->count()){
            return admin_abort('id为'.$id.'的标签不存在',403);
        }
        $data = TopicTag::query()->where("id",$id)->first();
        return view("plugins.Topic.Tag.edit",['data' => $data]);
    }

    #[PostMapping(path:"/admin/topic/tag/edit")]
    public function edit_post(EditTagRequest $request,AvatarUpload $upload){
        $icon = false;
        $id = $request->input("id");
        $name = $request->input("name");
        $description = $request->input("description");
        $color = $request->input("color");
        if($request->hasFile("icon")){
            $icon = true;
        }

        if($icon===true){
            // 上传icon
            $url = $upload->save($request->file("icon"),"admin",Str::random())['path'];
            TopicTag::query()->where("id",$id)->update([
                "name" => $name,
                "description" => $description,
                "color" => $color,
                "icon" => $url
            ]);
        }else{
            TopicTag::query()->where("id",$id)->update([
                "name" => $name,
                "description" => $description,
                "color" => $color,
            ]);
        }
        return redirect()->back()->with("success","修改成功!")->go();

    }

    #[PostMapping(path:"/admin/topic/tag/remove")]
    public function remove(){
        $id = request()->input("id");
        if(!$id){
            return Json_Api(403,false,['msg' => '请求id不能为空']);
        }
        if($id===1){
            return Json_Api(403,false,['msg' => '安全起见,你不能删除id为1的标签,因为这属于是帖子的默认分类']);
        }
        if (!TopicTag::query()->where("id",$id)->count()){
            return Json_Api(403,false,['msg' => 'id为'.$id."的标签不存在"]);
        }
        TopicTag::query()->where("id",$id)->delete();
        return Json_Api(200,true,['msg' => '删除成功!']);
    }
}